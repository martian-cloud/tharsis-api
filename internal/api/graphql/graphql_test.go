package graphql

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	complexity "gitlab.com/infor-cloud/martian-cloud/tharsis/graphql-query-complexity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/resolver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/schema"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

type panickingResolver struct{}

func (*panickingResolver) Boom() string {
	panic("raw panic detail that must not leak")
}

func TestHTTPHandler_ServeHTTP_PanicSanitizedAndLogged(t *testing.T) {
	log, observed := logger.NewForTest()

	schema := graphql.MustParseSchema(
		`schema { query: Query } type Query { boom: String! }`,
		&panickingResolver{},
		graphql.UseFieldResolvers(),
		graphql.Logger(&panicLogger{logger: log}),
		graphql.PanicHandler(&panicHandler{}),
	)

	complexityOpts := complexity.DefaultOptions()
	calculator, err := complexity.NewCalculator(schema.AST(), &complexityOpts)
	require.NoError(t, err)

	h := &httpHandler{
		schema: schema,
		logger: log,
		ctxGenerator: &contextGenerator{
			resolverState: &resolver.State{Logger: log},
			loaders:       loader.NewCollection(),
		},
		queryComplexityCalculator: calculator,
		maxGraphqlComplexity:      0, // disables rate limiting; the rate-limit store is never touched
		maxBodySize:               1 << 20,
	}

	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(`{"query":"{ boom }"}`))
	req = req.WithContext(auth.WithSubject(context.Background(), "test-subject"))
	rec := httptest.NewRecorder()

	require.NotPanics(t, func() { h.ServeHTTP(rec, req) })

	body := rec.Body.String()
	// Client gets the sanitized message, not the raw panic text.
	assert.Contains(t, body, errors.InternalErrorMessage)
	assert.NotContains(t, body, "raw panic detail that must not leak")
	assert.Contains(t, body, "INTERNAL_SERVER_ERROR")

	// The panic is logged at ERROR with the value and stack as structured fields.
	entries := observed.All()
	require.NotEmpty(t, entries)
	entry := entries[0]
	assert.Equal(t, "error", entry.Level.String())
	assert.Equal(t, "graphql panic recovered", entry.Message)
	fields := entry.ContextMap()
	assert.Equal(t, "raw panic detail that must not leak", fields["panic"])
	assert.Contains(t, fields["stack"], "goroutine")
}

var (
	fuzzCalculatorOnce sync.Once
	fuzzCalculator     *complexity.Calculator
	fuzzCalculatorErr  error
)

// newFuzzComplexityCalculator returns a calculator over the real schema, configured the way
// NewGraphQL configures the handler's. It is built once because parsing the schema dominates the
// cost of a fuzz iteration otherwise.
func newFuzzComplexityCalculator(t *testing.T) *complexity.Calculator {
	fuzzCalculatorOnce.Do(func() {
		schemaStr, err := schema.String()
		if err != nil {
			fuzzCalculatorErr = err
			return
		}
		parsed, err := graphql.ParseSchema(schemaStr, resolver.NewRootResolver(), graphql.UseFieldResolvers())
		if err != nil {
			fuzzCalculatorErr = err
			return
		}
		opts := complexity.DefaultOptions()
		opts.MaxNestedLevels = maxQueryNestedLevels
		fuzzCalculator, fuzzCalculatorErr = complexity.NewCalculator(parsed.ASTSchema(), &opts)
	})
	require.NoError(t, fuzzCalculatorErr)
	return fuzzCalculator
}

// FuzzQueryComplexityLimits builds queries against the real schema from a fuzzed byte stream
// (nesting through fields and connections, aliases and field repetition, inline fragments, named
// fragments with spreads and cycles, first/last page sizes as literals or variables) and checks
// that every query the complexity guard accepts is within its limits as computed independently.
// The same bytes are also fed to the guard as raw query text, which must never panic.
//
// Fragments may be spread any number of times and page sizes may be as large as an int holds:
// the calculator expands a fragment once per selection set, bounds the selections it visits, and
// saturates the cost rather than letting it wrap around, so the cost is never negative. A fuzz
// input that makes the calculator slow is a regression of the first two, such as the fragment
// chains seeded below that used to take minutes.
func FuzzQueryComplexityLimits(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("{ me { __typename } }"))
	f.Add([]byte(`query { group(fullPath: "a") { parent { parent { name } } } }`))
	f.Add([]byte(`{ groups(first: 2147483647) { edges { node { id } } } }`))
	f.Add([]byte(`{ groups(first: $n) { edges { node { id } } } }`))
	f.Add([]byte(`{ groups(first: 2147483647) { edges { node { descendentGroups(first: 2147483647) { edges { node { ` +
		`descendentGroups(first: 2147483647) { edges { node { id name } } } } } } } } } }`))
	f.Add([]byte(`{ group(fullPath: "a") { ...A } } fragment A on Group { parent { ...A } }`))
	f.Add([]byte(`fragment A on Group { ...A } { group(fullPath: "a") { ... on Node { id } } }`))
	f.Add([]byte(`{ a(first: "x") { b } }`))
	f.Add([]byte(`{ a(first: 1.5) { b } }`))
	f.Add([]byte(`{ a( `))
	f.Add([]byte(`mutation { createGroup(input: {}) { group { id } } } subscription { a }`))
	f.Add([]byte(strings.Repeat("{ a ", 30) + strings.Repeat("}", 30)))
	f.Add(append([]byte{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}, 0))
	f.Add([]byte{3, 1, 0, 3, 1, 0, 3, 1, 0, 3, 1, 0, 3, 1, 0, 3, 1, 0, 3, 1, 0})
	// fragment F0 on Group { parent { ...F0 } }: a cycle through a field, infinitely deep.
	f.Add([]byte{0x01, 0, 5, 0, 0, 1, 0, 5, 0})
	// fragment F0 on Group { ...F0 name }: a cycle that adds nothing.
	f.Add([]byte{0x01, 0, 5, 0, 1, 5, 0, 0})
	// Nesting continued through two fragments.
	f.Add([]byte{0x02, 0, 5, 0, 0, 1, 0, 5, 1, 0, 2, 0, 2, 0, 2, 0, 2, 0, 2, 0, 2, 0, 1, 0, 1, 0, 1})
	// 151 aliased fields in one object.
	f.Add([]byte{0x00, 0, 6, 150})
	f.Add([]byte{0x80 | 2, 7, 1, 6, 0, 2, 2, 7, 1, 2, 7, 0, 0})
	f.Add([]byte{0x80 | 3, 7, 0, 2, 7, 0, 6, 0, 2, 6, 1, 0, 2, 7, 2, 0})
	f.Add([]byte{0x40, 5, 150, 3, 4, 0, 4, 0, 0})
	f.Add([]byte{0, 3, 9, 3, 6, 3, 5, 5, 2, 4, 8, 1})
	// Each fragment spreads the next twice, which once expanded to 2^30 group fields.
	f.Add([]byte(fragmentChain(30, "Query", "...F%[1]d ...F%[1]d", `group(fullPath: "a") { id }`)))
	// The same through a field, with the merged sub-selections of every parent expanded again.
	f.Add([]byte(`{ group(fullPath: "a") { ...A } } fragment A on Group { parent { ...A ...A } }`))
	// Aliases multiplied through fragments, 60^6 selections within the nesting and per object
	// limits.
	f.Add([]byte(fragmentChain(6, "Group", strings.Repeat("a%[2]d: parent { ...F%[1]d } ", 60), "id")))
	// Many generated spreads of every fragment.
	f.Add([]byte{0x83, 0, 3, 5, 0, 5, 1, 5, 2, 3, 5, 3, 5, 2, 5, 1, 3, 1, 5, 2, 5, 2, 5, 3, 3, 1, 5, 3, 5, 3})

	f.Fuzz(func(t *testing.T, data []byte) {
		calculator := newFuzzComplexityCalculator(t)

		// Raw text: the guard must reject malformed input with an error, never a panic.
		raw := string(data)
		_, _ = calculator.Calculate(raw, nil)
		_, _ = calculator.Calculate(raw, map[string]any{"n": 1e300, "m": "x"})

		g := newQueryGen(data)
		doc := g.generate()
		cost, err := calculator.Calculate(doc.text, doc.variables)
		if err != nil {
			return
		}

		levels, maxFields := doc.measure(maxQueryNestedLevels + 1)
		assert.LessOrEqual(t, levels, maxQueryNestedLevels, "nesting limit bypassed by:\n%s", doc.text)
		assert.LessOrEqual(t, maxFields, complexity.DefaultMaxFieldsPerObject, "fields per object limit bypassed by:\n%s", doc.text)
		assert.GreaterOrEqual(t, cost, 0, "negative cost for:\n%s", doc.text)
	})
}

// fragmentChain returns a query spreading F0 on the root type's selections, where each of n
// fragments has the body format formats with its successor's index and, for aliases, a counter.
// The last fragment selects leaf.
func fragmentChain(n int, on, format, leaf string) string {
	var b strings.Builder
	if on == "Query" {
		b.WriteString("{ ...F0 }\n")
	} else {
		b.WriteString(`{ group(fullPath: "a") { ...F0 } }` + "\n")
	}
	for i := range n {
		fmt.Fprintf(&b, "fragment F%d on %s { ", i, on)
		// Every alias in the body takes the same index, so a counter keeps them distinct.
		body := format
		for j := 0; strings.Contains(body, "%[2]d"); j++ {
			body = strings.Replace(body, "%[2]d", strconv.Itoa(j), 1)
		}
		fmt.Fprintf(&b, body, i+1)
		b.WriteString(" }\n")
	}
	fmt.Fprintf(&b, "fragment F%d on %s { %s }\n", n, on, leaf)
	return b.String()
}

// selKind is the kind of a generated selection.
type selKind int

const (
	// selLeaf is a scalar field, possibly aliased.
	selLeaf selKind = iota
	// selField is an object field with a selection set of the same type (Group.parent).
	selField
	// selConnection is a paginated connection whose nodes are of the same type
	// (Group.descendentGroups { edges { node { ... } } }), three levels deep.
	selConnection
	// selInline is an inline fragment on the type being selected, which merges into it.
	selInline
	// selCondition is an inline fragment on an interface, charged as a separate condition.
	selCondition
	// selSpread is a named fragment spread.
	selSpread
	// selRepeat is a run of aliased leaves, for the per object field limit.
	selRepeat
)

// sel is one generated selection on the Group type.
type sel struct {
	kind     selKind
	alias    string
	count    int    // number of leaves for selRepeat
	frag     int    // fragment index for selSpread
	pageArg  string // first or last, for selConnection
	page     string // the page size literal or variable, for selConnection
	children []*sel
}

// genDoc is a generated document: its text, its variables, and the structure it was built from.
type genDoc struct {
	text      string
	variables map[string]any
	root      []*sel // selections of the root group field
	fragments [][]*sel
}

// queryGen turns a byte stream into a query. Running out of bytes yields zeros, which select
// leaves, so generation always terminates.
type queryGen struct {
	data      []byte
	pos       int
	fragments [][]*sel
	useVar    bool
	nodeCount int
}

// pageSizes are the page sizes a connection can ask for: valid ones, the resolver's maximum and
// one past it, zero, a negative one, and ones large enough that nesting them overflows an int.
var pageSizes = []int{0, 1, 2, 10, 100, 101, -1, math.MaxInt32, math.MaxInt}

func newQueryGen(data []byte) *queryGen {
	return &queryGen{data: data}
}

func (g *queryGen) next() byte {
	if g.pos >= len(g.data) {
		return 0
	}
	b := g.data[g.pos]
	g.pos++
	return b
}

// generate builds the document. The first byte holds the number of fragments (0x03 plus 0x80)
// and, in 0x40, whether page sizes are passed as a variable, whose value bits 3-5 select.
func (g *queryGen) generate() *genDoc {
	header := g.next()
	numFragments := int(header&0x03) + int(header>>7)
	g.useVar = header&0x40 != 0
	pageSize := pageSizes[int(header>>3&0x07)%len(pageSizes)]
	g.fragments = make([][]*sel, numFragments)

	root := g.selections(0, -1)
	for i := range g.fragments {
		g.fragments[i] = g.selections(0, i)
	}

	doc := &genDoc{root: root, fragments: g.fragments}
	var b strings.Builder
	if g.useVar {
		b.WriteString("query Q($n: Int) ")
		doc.variables = map[string]any{"n": float64(pageSize)}
	}
	b.WriteString(`{ group(fullPath: "g") `)
	writeSelections(&b, root)
	b.WriteString(" }")
	for i, frag := range g.fragments {
		fmt.Fprintf(&b, "\nfragment F%d on Group ", i)
		writeSelections(&b, frag)
	}
	doc.text = b.String()
	return doc
}

// selections generates a selection set. inFragment is the index of the fragment whose body is
// being generated, or -1 for the operation itself. depth only bounds the generator so that it
// terminates; it is deliberately allowed past the nesting limit.
func (g *queryGen) selections(depth, inFragment int) []*sel {
	n := 1 + int(g.next()%4)
	sels := make([]*sel, 0, n)
	for range n {
		sels = append(sels, g.selection(depth, inFragment))
	}
	return sels
}

func (g *queryGen) selection(depth, inFragment int) *sel {
	g.nodeCount++
	b := g.next()
	// Keep documents bounded: past this point only leaves are generated.
	if depth > 3*maxQueryNestedLevels || g.nodeCount > 400 {
		b = 0
	}
	alias := ""
	if b&0x80 != 0 {
		alias = fmt.Sprintf("a%d", b>>4&0x07)
	}

	switch selKind((b & 0x07) % 7) {
	case selField:
		return &sel{kind: selField, alias: alias, children: g.selections(depth+1, inFragment)}
	case selConnection:
		conn := &sel{kind: selConnection, alias: alias, pageArg: "first", page: "$n"}
		if b&0x08 != 0 {
			conn.pageArg = "last"
		}
		if !g.useVar {
			conn.page = strconv.Itoa(pageSizes[int(g.next())%len(pageSizes)])
		}
		conn.children = g.selections(depth+3, inFragment)
		return conn
	case selInline:
		return &sel{kind: selInline, children: g.selections(depth, inFragment)}
	case selCondition:
		return &sel{kind: selCondition}
	case selSpread:
		if len(g.fragments) == 0 {
			break
		}
		return &sel{kind: selSpread, frag: int(g.next()) % len(g.fragments)}
	case selRepeat:
		return &sel{kind: selRepeat, count: 1 + int(g.next())}
	}
	return &sel{kind: selLeaf, alias: alias}
}

func writeSelections(b *strings.Builder, sels []*sel) {
	b.WriteString("{ ")
	for _, s := range sels {
		writeSelection(b, s)
		b.WriteString(" ")
	}
	b.WriteString("}")
}

func writeSelection(b *strings.Builder, s *sel) {
	if s.alias != "" {
		b.WriteString(s.alias + ": ")
	}
	switch s.kind {
	case selLeaf:
		b.WriteString("name")
	case selField:
		b.WriteString("parent ")
		writeSelections(b, s.children)
	case selConnection:
		fmt.Fprintf(b, "descendentGroups(%s: %s) { pageInfo { hasNextPage } edges { node ", s.pageArg, s.page)
		writeSelections(b, s.children)
		b.WriteString(" } }")
	case selInline:
		b.WriteString("... on Group ")
		writeSelections(b, s.children)
	case selCondition:
		b.WriteString("... on Node { id }")
	case selSpread:
		fmt.Fprintf(b, "...F%d", s.frag)
	case selRepeat:
		for i := range s.count {
			fmt.Fprintf(b, "r%d: id ", i)
		}
	}
}

// measure returns the deepest selection set level the document reaches (the root operation's
// set being level 1) and the most distinct response keys any single selection set resolves,
// expanding fragments as execution would. A cycle through a field is infinitely deep, so the walk
// stops once it passes limit.
func (d *genDoc) measure(limit int) (levels, maxFields int) {
	w := &measurer{doc: d, limit: limit}
	// The root operation's set holds the group field, whose own set is level 2.
	w.levels = 1
	w.maxFields = 1
	w.walk(d.root, 2)
	return w.levels, w.maxFields
}

type measurer struct {
	doc       *genDoc
	limit     int
	levels    int
	maxFields int
}

// walk measures one selection set at the given level.
func (w *measurer) walk(sels []*sel, level int) {
	if level > w.levels {
		w.levels = level
	}
	if level > w.limit {
		return
	}
	set := &measuredSet{keys: map[string]bool{}, children: map[childKey][]*sel{}, spread: map[int]bool{}}
	w.collect(sels, set)
	if len(set.keys) > w.maxFields {
		w.maxFields = len(set.keys)
	}
	for _, key := range set.order {
		children := set.children[key]
		if key.kind == selConnection {
			// connection { pageInfo, edges } then edges { node } then node { children }.
			for _, l := range []int{level + 1, level + 2} {
				if l > w.levels {
					w.levels = l
				}
			}
			w.walk(children, level+3)
		} else {
			w.walk(children, level+1)
		}
	}
}

// measuredSet is one selection set flattened the way execution collects it.
type measuredSet struct {
	// keys are the response keys the set resolves.
	keys map[string]bool
	// children merges the sub-selections of every field sharing a response key, so each is walked
	// once however many times the field is selected. Fields are told apart by kind as well, since
	// a connection's children sit deeper than a plain field's.
	children map[childKey][]*sel
	order    []childKey
	// spread names the fragments already expanded into this set. Spreading one again adds nothing.
	spread map[int]bool
}

type childKey struct {
	responseKey string
	kind        selKind
}

// collect adds the response keys of a selection set and the sub-selections of its fields, merging
// fragments on the same type into it.
func (w *measurer) collect(sels []*sel, set *measuredSet) {
	key := func(s *sel, name string) string {
		if s.alias != "" {
			return s.alias
		}
		return name
	}
	addChildren := func(s *sel, name string) {
		k := childKey{responseKey: key(s, name), kind: s.kind}
		set.keys[k.responseKey] = true
		if _, ok := set.children[k]; !ok {
			set.order = append(set.order, k)
		}
		set.children[k] = append(set.children[k], s.children...)
	}
	for _, s := range sels {
		switch s.kind {
		case selLeaf:
			set.keys[key(s, "name")] = true
		case selField:
			addChildren(s, "parent")
		case selConnection:
			addChildren(s, "descendentGroups")
		case selInline:
			w.collect(s.children, set)
		case selCondition:
			// Charged as its own set by the guard, at least as deep as this one.
		case selSpread:
			if set.spread[s.frag] {
				continue
			}
			set.spread[s.frag] = true
			w.collect(w.doc.fragments[s.frag], set)
		case selRepeat:
			for i := range s.count {
				set.keys[fmt.Sprintf("r%d", i)] = true
			}
		}
	}
}
