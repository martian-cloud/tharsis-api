/**
 * @generated SignedSource<<2c059525d65cb93248aef0febb832e8c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type JobLogsQuery$variables = {
  id: string;
  limit: number;
  startOffset: number;
};
export type JobLogsQuery$data = {
  readonly job: {
    readonly id: string;
    readonly logLastUpdatedAt: any | null | undefined;
    readonly logSize: number;
    readonly logs: string;
  } | null | undefined;
};
export type JobLogsQuery = {
  response: JobLogsQuery$data;
  variables: JobLogsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "id"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "limit"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "startOffset"
},
v3 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "id",
        "variableName": "id"
      }
    ],
    "concreteType": "Job",
    "kind": "LinkedField",
    "name": "job",
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
        "name": "logLastUpdatedAt",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "logSize",
        "storageKey": null
      },
      {
        "alias": null,
        "args": [
          {
            "kind": "Variable",
            "name": "limit",
            "variableName": "limit"
          },
          {
            "kind": "Variable",
            "name": "startOffset",
            "variableName": "startOffset"
          }
        ],
        "kind": "ScalarField",
        "name": "logs",
        "storageKey": null
      }
    ],
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "JobLogsQuery",
    "selections": (v3/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v2/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Operation",
    "name": "JobLogsQuery",
    "selections": (v3/*: any*/)
  },
  "params": {
    "cacheID": "c869eea56332c972ff46e2dda19752d3",
    "id": null,
    "metadata": {},
    "name": "JobLogsQuery",
    "operationKind": "query",
    "text": "query JobLogsQuery(\n  $id: String!\n  $startOffset: Int!\n  $limit: Int!\n) {\n  job(id: $id) {\n    id\n    logLastUpdatedAt\n    logSize\n    logs(startOffset: $startOffset, limit: $limit)\n  }\n}\n"
  }
};
})();

(node as any).hash = "eb3f03a274c9336360f7d2c58af5e959";

export default node;
