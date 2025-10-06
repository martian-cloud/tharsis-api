/**
 * @generated SignedSource<<d6e45c3a783eb4dac839923e8fe6e047>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type HomeQuery$variables = Record<PropertyKey, never>;
export type HomeQuery$data = {
  readonly " $fragmentSpreads": FragmentRefs<"HomeFragment_activity">;
};
export type HomeQuery = {
  response: HomeQuery$data;
  variables: HomeQuery$variables;
};

const node: ConcreteRequest = {
  "fragment": {
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "HomeQuery",
    "selections": [
      {
        "args": null,
        "kind": "FragmentSpread",
        "name": "HomeFragment_activity"
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "HomeQuery",
    "selections": [
      {
        "alias": null,
        "args": [
          {
            "kind": "Literal",
            "name": "first",
            "value": 0
          }
        ],
        "concreteType": "ActivityEventConnection",
        "kind": "LinkedField",
        "name": "activityEvents",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "totalCount",
            "storageKey": null
          }
        ],
        "storageKey": "activityEvents(first:0)"
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "Config",
        "kind": "LinkedField",
        "name": "config",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "tharsisSupportUrl",
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "ee0ff17b31b316ea80b53cf2c357d61e",
    "id": null,
    "metadata": {},
    "name": "HomeQuery",
    "operationKind": "query",
    "text": "query HomeQuery {\n  ...HomeFragment_activity\n}\n\nfragment HomeFragment_activity on Query {\n  activityEvents(first: 0) {\n    totalCount\n  }\n  config {\n    tharsisSupportUrl\n  }\n}\n"
  }
};

(node as any).hash = "6737c254c3859d4c50878c0ec24d5371";

export default node;
