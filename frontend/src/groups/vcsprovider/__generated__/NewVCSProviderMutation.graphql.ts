/**
 * @generated SignedSource<<2c9d40c5b6d1f0beba88da087c4da516>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type VCSProviderType = "github" | "gitlab" | "%future added value";
export type CreateVCSProviderInput = {
  autoCreateWebhooks: boolean;
  clientMutationId?: string | null | undefined;
  description: string;
  groupId?: string | null | undefined;
  groupPath?: string | null | undefined;
  name: string;
  oAuthClientId: string;
  oAuthClientSecret: string;
  type: VCSProviderType;
  url?: string | null | undefined;
};
export type NewVCSProviderMutation$variables = {
  connections: ReadonlyArray<string>;
  input: CreateVCSProviderInput;
};
export type NewVCSProviderMutation$data = {
  readonly createVCSProvider: {
    readonly oAuthAuthorizationUrl: string;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly vcsProvider: {
      readonly description: string;
      readonly id: string;
      readonly name: string;
    } | null | undefined;
  };
};
export type NewVCSProviderMutation = {
  response: NewVCSProviderMutation$data;
  variables: NewVCSProviderMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "connections"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "input"
},
v2 = [
  {
    "kind": "Variable",
    "name": "input",
    "variableName": "input"
  }
],
v3 = {
  "alias": null,
  "args": null,
  "concreteType": "VCSProvider",
  "kind": "LinkedField",
  "name": "vcsProvider",
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
      "kind": "ScalarField",
      "name": "name",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "description",
      "storageKey": null
    }
  ],
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "oAuthAuthorizationUrl",
  "storageKey": null
},
v5 = {
  "alias": null,
  "args": null,
  "concreteType": "Problem",
  "kind": "LinkedField",
  "name": "problems",
  "plural": true,
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "message",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "field",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "type",
      "storageKey": null
    }
  ],
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "NewVCSProviderMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateVCSProviderPayload",
        "kind": "LinkedField",
        "name": "createVCSProvider",
        "plural": false,
        "selections": [
          (v3/*: any*/),
          (v4/*: any*/),
          (v5/*: any*/)
        ],
        "storageKey": null
      }
    ],
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v1/*: any*/),
      (v0/*: any*/)
    ],
    "kind": "Operation",
    "name": "NewVCSProviderMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateVCSProviderPayload",
        "kind": "LinkedField",
        "name": "createVCSProvider",
        "plural": false,
        "selections": [
          (v3/*: any*/),
          {
            "alias": null,
            "args": null,
            "filters": null,
            "handle": "prependNode",
            "key": "",
            "kind": "LinkedHandle",
            "name": "vcsProvider",
            "handleArgs": [
              {
                "kind": "Variable",
                "name": "connections",
                "variableName": "connections"
              },
              {
                "kind": "Literal",
                "name": "edgeTypeName",
                "value": "VCSProviderEdge"
              }
            ]
          },
          (v4/*: any*/),
          (v5/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "5ecf499c54abdc85cb2323c860346948",
    "id": null,
    "metadata": {},
    "name": "NewVCSProviderMutation",
    "operationKind": "mutation",
    "text": "mutation NewVCSProviderMutation(\n  $input: CreateVCSProviderInput!\n) {\n  createVCSProvider(input: $input) {\n    vcsProvider {\n      id\n      name\n      description\n    }\n    oAuthAuthorizationUrl\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "bb8e848e2739c2cbbb211b2374a94600";

export default node;
