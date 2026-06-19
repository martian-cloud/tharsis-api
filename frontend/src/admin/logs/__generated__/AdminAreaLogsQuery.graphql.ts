/**
 * @generated SignedSource<<85866f3834b4ae6ed3b95c5506e1a50e>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type AdminLogTailLevel = "DEBUG" | "ERROR" | "INFO" | "WARN" | "%future added value";
export type AdminAreaLogsQuery$variables = {
  levels?: ReadonlyArray<AdminLogTailLevel> | null | undefined;
  limit?: number | null | undefined;
  search?: string | null | undefined;
};
export type AdminAreaLogsQuery$data = {
  readonly adminLogTail: ReadonlyArray<{
    readonly caller: string | null | undefined;
    readonly fields: string | null | undefined;
    readonly id: string;
    readonly level: AdminLogTailLevel;
    readonly message: string;
    readonly stack: string | null | undefined;
    readonly timestamp: any;
  }>;
  readonly config: {
    readonly adminLogTailStorePluginType: string;
  };
};
export type AdminAreaLogsQuery = {
  response: AdminAreaLogsQuery$data;
  variables: AdminAreaLogsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "levels"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "limit"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "search"
},
v3 = [
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
        "name": "adminLogTailStorePluginType",
        "storageKey": null
      }
    ],
    "storageKey": null
  },
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "levels",
        "variableName": "levels"
      },
      {
        "kind": "Variable",
        "name": "limit",
        "variableName": "limit"
      },
      {
        "kind": "Variable",
        "name": "search",
        "variableName": "search"
      }
    ],
    "concreteType": "AdminLogTailEntry",
    "kind": "LinkedField",
    "name": "adminLogTail",
    "plural": true,
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
        "name": "timestamp",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "level",
        "storageKey": null
      },
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
        "name": "caller",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "stack",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "fields",
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
    "name": "AdminAreaLogsQuery",
    "selections": (v3/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v1/*: any*/),
      (v0/*: any*/),
      (v2/*: any*/)
    ],
    "kind": "Operation",
    "name": "AdminAreaLogsQuery",
    "selections": (v3/*: any*/)
  },
  "params": {
    "cacheID": "2a3fa8a79fb629c40b4aba59dd600d97",
    "id": null,
    "metadata": {},
    "name": "AdminAreaLogsQuery",
    "operationKind": "query",
    "text": "query AdminAreaLogsQuery(\n  $limit: Int\n  $levels: [AdminLogTailLevel!]\n  $search: String\n) {\n  config {\n    adminLogTailStorePluginType\n  }\n  adminLogTail(limit: $limit, levels: $levels, search: $search) {\n    id\n    timestamp\n    level\n    message\n    caller\n    stack\n    fields\n  }\n}\n"
  }
};
})();

(node as any).hash = "5f14d1290457028cccf45f5e8763d9a6";

export default node;
