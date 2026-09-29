/**
 * @generated SignedSource<<507f2abf8abac2c6f4a5f893a3687b62>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type GroupSettingsQuery$variables = {
  id: string;
};
export type GroupSettingsQuery$data = {
  readonly node: {
    readonly " $fragmentSpreads": FragmentRefs<"GroupSettingsContentFragment_group">;
  } | null | undefined;
};
export type GroupSettingsQuery = {
  response: GroupSettingsQuery$data;
  variables: GroupSettingsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "id"
  }
],
v1 = [
  {
    "kind": "Variable",
    "name": "id",
    "variableName": "id"
  }
],
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "inherited",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "namespacePath",
  "storageKey": null
},
v4 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "value",
  "storageKey": null
},
v5 = [
  (v2/*: any*/),
  (v4/*: any*/),
  (v3/*: any*/)
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "GroupSettingsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
        "plural": false,
        "selections": [
          {
            "kind": "InlineFragment",
            "selections": [
              {
                "args": null,
                "kind": "FragmentSpread",
                "name": "GroupSettingsContentFragment_group"
              }
            ],
            "type": "Group",
            "abstractKey": null
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
    "name": "GroupSettingsQuery",
    "selections": [
      {
        "alias": null,
        "args": (v1/*: any*/),
        "concreteType": null,
        "kind": "LinkedField",
        "name": "node",
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
                "args": null,
                "kind": "ScalarField",
                "name": "fullPath",
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceRunnerTags",
                "kind": "LinkedField",
                "name": "runnerTags",
                "plural": false,
                "selections": [
                  (v2/*: any*/),
                  (v3/*: any*/),
                  (v4/*: any*/)
                ],
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceDriftDetectionEnabled",
                "kind": "LinkedField",
                "name": "driftDetectionEnabled",
                "plural": false,
                "selections": (v5/*: any*/),
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceProviderMirrorEnabled",
                "kind": "LinkedField",
                "name": "providerMirrorEnabled",
                "plural": false,
                "selections": (v5/*: any*/),
                "storageKey": null
              },
              {
                "alias": null,
                "args": null,
                "concreteType": "NamespaceOutputVisibility",
                "kind": "LinkedField",
                "name": "outputVisibility",
                "plural": false,
                "selections": (v5/*: any*/),
                "storageKey": null
              }
            ],
            "type": "Group",
            "abstractKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "id",
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ]
  },
  "params": {
    "cacheID": "186afac61a7202bedb43650b4afea52b",
    "id": null,
    "metadata": {},
    "name": "GroupSettingsQuery",
    "operationKind": "query",
    "text": "query GroupSettingsQuery(\n  $id: String!\n) {\n  node(id: $id) {\n    __typename\n    ... on Group {\n      ...GroupSettingsContentFragment_group\n    }\n    id\n  }\n}\n\nfragment DriftDetectionSettingsFormFragment_driftDetectionEnabled on NamespaceDriftDetectionEnabled {\n  inherited\n  namespacePath\n  value\n}\n\nfragment GroupAdvancedSettingsFragment_group on Group {\n  name\n  fullPath\n  ...MigrateGroupDialogFragment_group\n}\n\nfragment GroupDriftDetectionSettingsFragment_group on Group {\n  fullPath\n  driftDetectionEnabled {\n    inherited\n    value\n    ...DriftDetectionSettingsFormFragment_driftDetectionEnabled\n  }\n}\n\nfragment GroupGeneralSettingsFragment_group on Group {\n  name\n  description\n  fullPath\n}\n\nfragment GroupOutputVisibilitySettingsFragment_group on Group {\n  fullPath\n  outputVisibility {\n    inherited\n    value\n    ...OutputVisibilitySettingsFormFragment_outputVisibility\n  }\n}\n\nfragment GroupProviderMirrorSettingsFragment_group on Group {\n  fullPath\n  providerMirrorEnabled {\n    inherited\n    value\n    ...ProviderMirrorSettingsFormFragment_providerMirrorEnabled\n  }\n}\n\nfragment GroupRunnerSettingsFragment_group on Group {\n  fullPath\n  runnerTags {\n    inherited\n    namespacePath\n    value\n    ...RunnerSettingsForm_runnerTags\n  }\n}\n\nfragment GroupSettingsContentFragment_group on Group {\n  ...GroupGeneralSettingsFragment_group\n  ...GroupAdvancedSettingsFragment_group\n  ...GroupRunnerSettingsFragment_group\n  ...GroupDriftDetectionSettingsFragment_group\n  ...GroupProviderMirrorSettingsFragment_group\n  ...GroupOutputVisibilitySettingsFragment_group\n}\n\nfragment MigrateGroupDialogFragment_group on Group {\n  name\n  fullPath\n}\n\nfragment OutputVisibilitySettingsFormFragment_outputVisibility on NamespaceOutputVisibility {\n  inherited\n  namespacePath\n  value\n}\n\nfragment ProviderMirrorSettingsFormFragment_providerMirrorEnabled on NamespaceProviderMirrorEnabled {\n  inherited\n  namespacePath\n  value\n}\n\nfragment RunnerSettingsForm_runnerTags on NamespaceRunnerTags {\n  inherited\n  namespacePath\n  value\n}\n"
  }
};
})();

(node as any).hash = "5e36fa9538e2756eccd2bdeb859a8aa2";

export default node;
