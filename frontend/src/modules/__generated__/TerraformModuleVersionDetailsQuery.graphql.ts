/**
 * @generated SignedSource<<3b8c7ad82f90291f8619b09030b3e55b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionDetailsQuery$variables = {
  after?: string | null | undefined;
  first?: number | null | undefined;
  moduleName: string;
  registryNamespace: string;
  system: string;
  version?: string | null | undefined;
};
export type TerraformModuleVersionDetailsQuery$data = {
  readonly terraformModuleVersion: {
    readonly id: string;
    readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionDetailsIndexFragment_details">;
  } | null | undefined;
};
export type TerraformModuleVersionDetailsQuery = {
  response: TerraformModuleVersionDetailsQuery$data;
  variables: TerraformModuleVersionDetailsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "after"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "first"
},
v2 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "moduleName"
},
v3 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "registryNamespace"
},
v4 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "system"
},
v5 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "version"
},
v6 = [
  {
    "kind": "Variable",
    "name": "moduleName",
    "variableName": "moduleName"
  },
  {
    "kind": "Variable",
    "name": "registryNamespace",
    "variableName": "registryNamespace"
  },
  {
    "kind": "Variable",
    "name": "system",
    "variableName": "system"
  },
  {
    "kind": "Variable",
    "name": "version",
    "variableName": "version"
  }
],
v7 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
},
v8 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdAt",
  "storageKey": null
},
v9 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
},
v10 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "source",
  "storageKey": null
},
v11 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "type",
  "storageKey": null
},
v12 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "description",
  "storageKey": null
},
v13 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "sensitive",
  "storageKey": null
},
v14 = [
  (v9/*: any*/),
  (v11/*: any*/)
],
v15 = [
  {
    "kind": "Variable",
    "name": "after",
    "variableName": "after"
  },
  {
    "kind": "Variable",
    "name": "first",
    "variableName": "first"
  }
],
v16 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdBy",
  "storageKey": null
};
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/),
      (v2/*: any*/),
      (v3/*: any*/),
      (v4/*: any*/),
      (v5/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "TerraformModuleVersionDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v6/*: any*/),
        "concreteType": "TerraformModuleVersion",
        "kind": "LinkedField",
        "name": "terraformModuleVersion",
        "plural": false,
        "selections": [
          (v7/*: any*/),
          {
            "args": null,
            "kind": "FragmentSpread",
            "name": "TerraformModuleVersionDetailsIndexFragment_details"
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
    "argumentDefinitions": [
      (v3/*: any*/),
      (v2/*: any*/),
      (v4/*: any*/),
      (v5/*: any*/),
      (v1/*: any*/),
      (v0/*: any*/)
    ],
    "kind": "Operation",
    "name": "TerraformModuleVersionDetailsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v6/*: any*/),
        "concreteType": "TerraformModuleVersion",
        "kind": "LinkedField",
        "name": "terraformModuleVersion",
        "plural": false,
        "selections": [
          (v7/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "version",
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
              },
              (v8/*: any*/)
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "concreteType": "TerraformModule",
            "kind": "LinkedField",
            "name": "module",
            "plural": false,
            "selections": [
              (v7/*: any*/),
              (v9/*: any*/),
              (v10/*: any*/),
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "system",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "registryNamespace",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "private",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "repositoryUrl",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "groupPath",
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": [
              {
                "kind": "Literal",
                "name": "path",
                "value": "root"
              }
            ],
            "concreteType": "TerraformModuleConfigurationDetails",
            "kind": "LinkedField",
            "name": "configurationDetails",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "kind": "ScalarField",
                "name": "readme",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "TerraformModuleConfigurationDetailsVariable",
                "kind": "LinkedField",
                "name": "variables",
                "plural": true,
                "selections": [
                  (v9/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "required",
                    "storageKey": null
                  },
                  (v11/*: any*/),
                  (v12/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "default",
                    "storageKey": null
                  },
                  (v13/*: any*/)
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "TerraformModuleConfigurationDetailsOutput",
                "kind": "LinkedField",
                "name": "outputs",
                "plural": true,
                "selections": [
                  (v9/*: any*/),
                  (v12/*: any*/),
                  (v13/*: any*/)
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "TerraformModuleConfigurationDetailsResource",
                "kind": "LinkedField",
                "name": "managedResources",
                "plural": true,
                "selections": (v14/*: any*/),
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "TerraformModuleConfigurationDetailsResource",
                "kind": "LinkedField",
                "name": "dataResources",
                "plural": true,
                "selections": (v14/*: any*/),
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "TerraformModuleConfigurationDetailsProviderRequirement",
                "kind": "LinkedField",
                "name": "requiredProviders",
                "plural": true,
                "selections": [
                  (v10/*: any*/),
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "versionConstraints",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "storageKey": "configurationDetails(path:\"root\")"
          },
          {
            "alias": null,
            "args": (v15/*: any*/),
            "concreteType": "TerraformModuleAttestationConnection",
            "kind": "LinkedField",
            "name": "attestations",
            "plural": false,
            "selections": [
              {
                "alias": null,
                "args": null,
                "concreteType": "TerraformModuleAttestationEdge",
                "kind": "LinkedField",
                "name": "edges",
                "plural": true,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "concreteType": "TerraformModuleAttestation",
                    "kind": "LinkedField",
                    "name": "node",
                    "plural": false,
                    "selections": [
                      (v7/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "data",
                        "storageKey": null
                      },
                      (v12/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "predicateType",
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
                          (v8/*: any*/)
                        ],
                        "storageKey": null
                      },
                      (v16/*: any*/),
                      {
                        "alias": null,
                        "args": null,
                        "kind": "ScalarField",
                        "name": "__typename",
                        "storageKey": null
                      }
                    ],
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "cursor",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "PageInfo",
                "kind": "LinkedField",
                "name": "pageInfo",
                "plural": false,
                "selections": [
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "endCursor",
                    "storageKey": null
                  },
                  {
                    "alias": null,
                    "args": null,
                    "kind": "ScalarField",
                    "name": "hasNextPage",
                    "storageKey": null
                  }
                ],
                "storageKey": null
              }
            ],
            "storageKey": null
          },
          {
            "alias": null,
            "args": (v15/*: any*/),
            "filters": null,
            "handle": "connection",
            "key": "TerraformModuleVersionAttestList_attestations",
            "kind": "LinkedHandle",
            "name": "attestations"
          },
          (v16/*: any*/),
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "latest",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "shaSum",
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "27a8659f520f823c6aac17d44ef1956e",
    "id": null,
    "metadata": {},
    "name": "TerraformModuleVersionDetailsQuery",
    "operationKind": "query",
    "text": "query TerraformModuleVersionDetailsQuery(\n  $registryNamespace: String!\n  $moduleName: String!\n  $system: String!\n  $version: String\n  $first: Int\n  $after: String\n) {\n  terraformModuleVersion(registryNamespace: $registryNamespace, moduleName: $moduleName, system: $system, version: $version) {\n    id\n    ...TerraformModuleVersionDetailsIndexFragment_details\n  }\n}\n\nfragment TerraformModuleVersionAttestListFragment_attestations on TerraformModuleVersion {\n  id\n  attestations(first: $first, after: $after) {\n    edges {\n      node {\n        id\n        data\n        ...TerraformModuleVersionAttestListItemFragment_module\n        __typename\n      }\n      cursor\n    }\n    pageInfo {\n      endCursor\n      hasNextPage\n    }\n  }\n}\n\nfragment TerraformModuleVersionAttestListItemFragment_module on TerraformModuleAttestation {\n  id\n  description\n  predicateType\n  data\n  metadata {\n    createdAt\n  }\n  createdBy\n}\n\nfragment TerraformModuleVersionDetailsIndexFragment_details on TerraformModuleVersion {\n  id\n  version\n  status\n  metadata {\n    trn\n  }\n  module {\n    id\n    name\n    source\n    system\n    registryNamespace\n    private\n    ...TerraformModuleVersionListFragment_module\n  }\n  configurationDetails(path: \"root\") {\n    ...TerraformModuleVersionDocsFragment_configurationDetails\n  }\n  ...TerraformModuleVersionAttestListFragment_attestations\n  ...TerraformModuleVersionDetailsSidebarFragment_details\n}\n\nfragment TerraformModuleVersionDetailsSidebarFragment_details on TerraformModuleVersion {\n  id\n  version\n  createdBy\n  latest\n  shaSum\n  metadata {\n    createdAt\n  }\n  module {\n    id\n    name\n    system\n    registryNamespace\n    private\n    repositoryUrl\n    groupPath\n  }\n}\n\nfragment TerraformModuleVersionDocsDataSourcesFragment_dataResources on TerraformModuleConfigurationDetails {\n  dataResources {\n    name\n    type\n  }\n}\n\nfragment TerraformModuleVersionDocsFragment_configurationDetails on TerraformModuleConfigurationDetails {\n  readme\n  ...TerraformModuleVersionDocsSidebarFragment_configurationDetails\n  ...TerraformModuleVersionDocsInputsFragment_variables\n  ...TerraformModuleVersionDocsOutputsFragment_outputs\n  ...TerraformModuleVersionDocsResourcesFragment_managedResources\n  ...TerraformModuleVersionDocsDataSourcesFragment_dataResources\n  ...TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders\n}\n\nfragment TerraformModuleVersionDocsInputCardFragment_variable on TerraformModuleConfigurationDetailsVariable {\n  name\n  type\n  description\n  default\n  required\n  sensitive\n}\n\nfragment TerraformModuleVersionDocsInputsFragment_variables on TerraformModuleConfigurationDetails {\n  variables {\n    required\n    name\n    ...TerraformModuleVersionDocsInputCardFragment_variable\n  }\n}\n\nfragment TerraformModuleVersionDocsOutputsFragment_outputs on TerraformModuleConfigurationDetails {\n  outputs {\n    name\n    description\n    sensitive\n  }\n}\n\nfragment TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders on TerraformModuleConfigurationDetails {\n  requiredProviders {\n    source\n    versionConstraints\n  }\n}\n\nfragment TerraformModuleVersionDocsResourcesFragment_managedResources on TerraformModuleConfigurationDetails {\n  managedResources {\n    name\n    type\n  }\n}\n\nfragment TerraformModuleVersionDocsSidebarFragment_configurationDetails on TerraformModuleConfigurationDetails {\n  readme\n  variables {\n    name\n  }\n  outputs {\n    name\n  }\n  managedResources {\n    name\n  }\n  dataResources {\n    name\n  }\n  requiredProviders {\n    source\n  }\n}\n\nfragment TerraformModuleVersionListFragment_module on TerraformModule {\n  id\n}\n"
  }
};
})();

(node as any).hash = "a8488c66e7099cc94ca2237aaa16f04c";

export default node;
