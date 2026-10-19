/**
 * @generated SignedSource<<506c927da2454de469672f26704ecdb6>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupTreeContainerQuery$variables = Record<PropertyKey, never>;
export type GroupTreeContainerQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"GroupTreeContainerFragment_me">;
};
export type GroupTreeContainerQuery = {
  response: GroupTreeContainerQuery$data;
  variables: GroupTreeContainerQuery$variables;
};

const node: ConcreteRequest = {
  "fragment": {
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "GroupTreeContainerQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "GroupTreeContainerFragment_me"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "GroupTreeContainerQuery",
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
                "name": "adminModeEnabled",
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
    "cacheID": "336abdac908cf54d128e6bdd42ee6d01",
    "id": null,
    "metadata": {},
    "name": "GroupTreeContainerQuery",
    "operationKind": "query",
    "text": "query GroupTreeContainerQuery {\n  ...GroupTreeContainerFragment_me\n}\n\nfragment GroupTreeContainerFragment_me on Query {\n  me {\n    __typename\n    ... on User {\n      adminModeEnabled\n    }\n    ... on Node {\n      __isNode: __typename\n      id\n    }\n  }\n}\n"
  }
};

(node as any).hash = "2a9c5a7d689abc684c8b4215fc7b0503";

export default node;
