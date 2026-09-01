/**
 * @generated SignedSource<<478824f5694fc76a7aeeea897e0851d3>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type CleanupRuleKind = "RUNS" | "TERRAFORM_MODULES" | "TERRAFORM_PROVIDERS" | "%future added value";
export type CleanupPolicyListQuery$variables = {
  namespacePath: string;
};
export type CleanupPolicyListQuery$data = {
  readonly namespace: {
    readonly __typename: string;
    readonly effectiveCleanupPolicies: ReadonlyArray<{
      readonly kind: CleanupRuleKind;
      readonly " $fragmentSpreads": FragmentRefs<"CleanupPolicyListItem_fields">;
    }>;
    readonly fullPath: string;
    readonly id: string;
  } | null | undefined;
};
export type CleanupPolicyListQuery = {
  response: CleanupPolicyListQuery$data;
  variables: CleanupPolicyListQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "namespacePath"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "fullPath",
    "variableName": "namespacePath"
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "__typename",
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
  "kind": "ScalarField",
  "name": "kind",
  "storageKey": null
},
v6 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "strategy",
  "storageKey": null
},
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "description",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "nameGlob",
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "versionGlob",
  "storageKey": null
},
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "deleteAfterDays",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "CleanupPolicyListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "namespace",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          (v3/*: any*/),
          (v4/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "CleanupPolicy",
            "kind": "LinkedField",
            "name": "effectiveCleanupPolicies",
            "plural": true,
            "selections": [
              (v5/*: any*/),
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "CleanupPolicyListItem_fields"
              }
            ],
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ],
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "CleanupPolicyListQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "namespace",
        "plural": false,
        "selections": [
          (v2/*: any*/),
          (v3/*: any*/),
          (v4/*: any*/),
          {
            "alias": null,
            "args": null,
            "concreteType": "CleanupPolicy",
            "kind": "LinkedField",
            "name": "effectiveCleanupPolicies",
            "plural": true,
            "selections": [
              (v5/*: any*/),
              (v2/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "namespacePath",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "disabled",
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
                    "name": "trn",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "lastSweepCompletedAt",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "TerraformModuleCleanupPolicyData",
                "kind": "LinkedField",
                "name": "terraformModulePolicyData",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "TerraformModuleCleanupRule",
                    "kind": "LinkedField",
                    "name": "rules",
                    "plural": true,
                    "selections": [
                      (v6/*: any*/),
                      (v7/*: any*/),
                      (v8/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "systemGlob",
                        "storageKey": null
                      },
                      (v9/*: any*/),
                      (v10/*: any*/)
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "TerraformProviderCleanupPolicyData",
                "kind": "LinkedField",
                "name": "terraformProviderPolicyData",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "TerraformProviderCleanupRule",
                    "kind": "LinkedField",
                    "name": "rules",
                    "plural": true,
                    "selections": [
                      (v6/*: any*/),
                      (v7/*: any*/),
                      (v8/*: any*/),
                      (v9/*: any*/),
                      (v10/*: any*/)
                    ],
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "RunCleanupPolicyData",
                "kind": "LinkedField",
                "name": "runPolicyData",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "RunCleanupRule",
                    "kind": "LinkedField",
                    "name": "rules",
                    "plural": true,
                    "selections": [
                      (v6/*: any*/),
                      (v7/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "speculative",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "assessment",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "status",
                        "storageKey": null
                      },
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "keepMin",
                        "storageKey": null
                      },
                      (v10/*: any*/)
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
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "9e6551ac1cb7b7185a45a0cd67048cd2",
    "id": null,
    "metadata": {},
    "name": "CleanupPolicyListQuery",
    "operationKind": "query",
    "text": "query CleanupPolicyListQuery(\n  $namespacePath: String!\n) {\n  namespace(fullPath: $namespacePath) {\n    id\n    __typename\n    fullPath\n    effectiveCleanupPolicies {\n      kind\n      ...CleanupPolicyListItem_fields\n      id\n    }\n  }\n}\n\nfragment CleanupPolicyListItem_fields on CleanupPolicy {\n  id\n  kind\n  namespacePath\n  disabled\n  metadata {\n    trn\n  }\n  lastSweepCompletedAt\n  terraformModulePolicyData {\n    rules {\n      strategy\n      description\n      nameGlob\n      systemGlob\n      versionGlob\n      deleteAfterDays\n    }\n  }\n  terraformProviderPolicyData {\n    rules {\n      strategy\n      description\n      nameGlob\n      versionGlob\n      deleteAfterDays\n    }\n  }\n  runPolicyData {\n    rules {\n      strategy\n      description\n      speculative\n      assessment\n      status\n      keepMin\n      deleteAfterDays\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "13f692433d986af0062056b66d61aa8e";

export default node;
