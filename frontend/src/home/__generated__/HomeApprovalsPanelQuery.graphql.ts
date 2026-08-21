/**
 * @generated SignedSource<<1a259a15d3851688a7081bd5bb9c2717>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type HomeApprovalsPanelQuery$variables = Record<PropertyKey, never>;
export type HomeApprovalsPanelQuery$data = {
  readonly runGatesAwaitingMyDecision: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
        readonly metadata: {
          readonly createdAt: any;
        };
      } | null | undefined;
    } | null | undefined> | null | undefined;
    readonly totalCount: number;
  };
};
export type HomeApprovalsPanelQuery = {
  response: HomeApprovalsPanelQuery$data;
  variables: HomeApprovalsPanelQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Literal",
        "name": "first",
        "value": 1
      },
      {
        "kind": "Literal",
        "name": "sort",
        "value": "CREATED_AT_DESC"
      }
    ],
    "concreteType": "RunGateConnection",
    "kind": "LinkedField",
    "name": "runGatesAwaitingMyDecision",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "totalCount",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "RunGateEdge",
        "kind": "LinkedField",
        "name": "edges",
        "plural": true,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "RunGate",
            "kind": "LinkedField",
            "name": "node",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "id",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "ResourceMetadata",
                "kind": "LinkedField",
                "name": "metadata",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "createdAt",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ],
    "storageKey": "runGatesAwaitingMyDecision(first:1,sort:\"CREATED_AT_DESC\")"
  }
];
return {
  "fragment": {
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "HomeApprovalsPanelQuery",
    "selections": (v0/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "HomeApprovalsPanelQuery",
    "selections": (v0/*: any*/)
  },
  "params": {
    "cacheID": "d91b65a15208ce45aa46b8d7cab29583",
    "id": null,
    "metadata": {},
    "name": "HomeApprovalsPanelQuery",
    "operationKind": "query",
    "text": "query HomeApprovalsPanelQuery {\n  runGatesAwaitingMyDecision(first: 1, sort: CREATED_AT_DESC) {\n    totalCount\n    edges {\n      node {\n        id\n        metadata {\n          createdAt\n        }\n      }\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "ec35abb6a78f9d45d8f4ac24d825b735";

export default node;
