/**
 * @generated SignedSource<<868b10f93aae5b04a2c0ffee3b88ba94>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type AdminAreaQuery$variables = Record<PropertyKey, never>;
export type AdminAreaQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"AdminAreaEntryPointFragment_me">;
};
export type AdminAreaQuery = {
  response: AdminAreaQuery$data;
  variables: AdminAreaQuery$variables;
};

const node: ConcreteRequest = {
  "fragment": {
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "AdminAreaQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "AdminAreaEntryPointFragment_me"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "AdminAreaQuery",
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": null,
        "kind": "LinkedField",
        "name": "me",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "__typename",
            "storageKey": null
          },
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "admin",
                "storageKey": null
              }
            ],
            "type": "User",
            "abstractKey": null
          },
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "id",
                "storageKey": null
              }
            ],
            "type": "Node",
            "abstractKey": "__isNode"
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "b473416c6574c12076772b9d61c0cb27",
    "id": null,
    "metadata": {},
    "name": "AdminAreaQuery",
    "operationKind": "query",
    "text": "query AdminAreaQuery {\n  ...AdminAreaEntryPointFragment_me\n}\n\nfragment AdminAreaEntryPointFragment_me on Query {\n  me {\n    __typename\n    ... on User {\n      admin\n    }\n    ... on Node {\n      __isNode: __typename\n      id\n    }\n  }\n}\n"
  }
};

(node as any).hash = "1dbe478395a0a209dc46506cbc284b4c";

export default node;
