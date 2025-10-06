/**
 * @generated SignedSource<<c7ef4dc34a14fdb1e5f9e011bb3e74c6>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type CreateGroupInput = {
  clientMutationId?: string | null | undefined;
  description: string;
  driftDetectionEnabled?: NamespaceDriftDetectionEnabledInput | null | undefined;
  name: string;
  parentId?: string | null | undefined;
  parentPath?: string | null | undefined;
  runnerTags?: NamespaceRunnerTagsInput | null | undefined;
};
export type NamespaceDriftDetectionEnabledInput = {
  enabled?: boolean | null | undefined;
  inherit: boolean;
};
export type NamespaceRunnerTagsInput = {
  inherit: boolean;
  tags?: ReadonlyArray<string> | null | undefined;
};
export type NewGroupMutation$variables = {
  connections: ReadonlyArray<string>;
  input: CreateGroupInput;
};
export type NewGroupMutation$data = {
  readonly createGroup: {
    readonly group: {
      readonly fullPath: string;
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"GroupListItemFragment_group">;
    } | null | undefined;
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
  };
};
export type NewGroupMutation = {
  response: NewGroupMutation$data;
  variables: NewGroupMutation$variables;
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
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "fullPath",
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
},
v6 = [
  {
    "kind": "Literal",
    "name": "first",
    "value": 0
  }
],
v7 = [
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "totalCount",
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "NewGroupMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateGroupPayload",
        "kind": "LinkedField",
        "name": "createGroup",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Group",
            "kind": "LinkedField",
            "name": "group",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              (v4/*: any*/),
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "GroupListItemFragment_group"
              }
            ],
            "storageKey": null
          },
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
    "name": "NewGroupMutation",
    "selections": [
      {
        "alias": null,
        "args": (v2/*: any*/),
        "concreteType": "CreateGroupPayload",
        "kind": "LinkedField",
        "name": "createGroup",
        "plural": false,
        "selections": [
          {
            "alias": null,
            "args": null,
            "concreteType": "Group",
            "kind": "LinkedField",
            "name": "group",
            "plural": false,
            "selections": [
              (v3/*: any*/),
              (v4/*: any*/),
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
                    "name": "updatedAt",
                    "storageKey": null
                  }
                ],
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
              },
              {
                "alias": null,
                "args": (v6/*: any*/),
                "concreteType": "GroupConnection",
                "kind": "LinkedField",
                "name": "descendentGroups",
                "plural": false,
                "selections": (v7/*: any*/),
                "storageKey": "descendentGroups(first:0)"
              },
              {
                "alias": null,
                "args": (v6/*: any*/),
                "concreteType": "WorkspaceConnection",
                "kind": "LinkedField",
                "name": "workspaces",
                "plural": false,
                "selections": (v7/*: any*/),
                "storageKey": "workspaces(first:0)"
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "filters": null,
            "handle": "prependNode",
            "key": "",
            "kind": "LinkedHandle",
            "name": "group",
            "handleArgs": [
              {
                "kind": "Variable",
                "name": "connections",
                "variableName": "connections"
              },
              {
                "kind": "Literal",
                "name": "edgeTypeName",
                "value": "GroupEdge"
              }
            ]
          },
          (v5/*: any*/)
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "6779638af0752e75c96b9e1e154f2642",
    "id": null,
    "metadata": {},
    "name": "NewGroupMutation",
    "operationKind": "mutation",
    "text": "mutation NewGroupMutation(\n  $input: CreateGroupInput!\n) {\n  createGroup(input: $input) {\n    group {\n      id\n      fullPath\n      ...GroupListItemFragment_group\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n\nfragment GroupListItemFragment_group on Group {\n  metadata {\n    updatedAt\n  }\n  id\n  name\n  description\n  fullPath\n  descendentGroups(first: 0) {\n    totalCount\n  }\n  workspaces(first: 0) {\n    totalCount\n  }\n}\n"
  }
};
})();

(node as any).hash = "313138341ed8dd0a7fa6466f46f39010";

export default node;
