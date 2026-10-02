/**
 * @generated SignedSource<<f9c47a48a77d189a0738e1761f26522b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceDetailsIndexFragment_workspace$data = {
  readonly assessment: {
    readonly hasDrift: boolean;
  } | null | undefined;
  readonly currentApplyRun: {
    readonly id: string;
  } | null | undefined;
  readonly currentStateVersion: {
    readonly id: string;
    readonly inventory: {
      readonly " $fragmentSpreads": FragmentRefs<"StateVersionCheckResultsFragment_checkResults" | "StateVersionDependenciesFragment_dependencies" | "StateVersionResourcesFragment_resources">;
    };
    readonly run: {
      readonly id: string;
      readonly " $fragmentSpreads": FragmentRefs<"StateVersionInputVariablesFragment_variables">;
    } | null | undefined;
    readonly " $fragmentSpreads": FragmentRefs<"StateVersionFileFragment_stateVersion" | "StateVersionOutputsFragment_outputs">;
  } | null | undefined;
  readonly description: string;
  readonly destroyed: boolean;
  readonly fullPath: string;
  readonly id: string;
  readonly labels: ReadonlyArray<{
    readonly key: string;
    readonly value: string;
  }>;
  readonly locked: boolean;
  readonly metadata: {
    readonly trn: string;
  };
  readonly name: string;
  readonly preventDestroyPlan: boolean;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceDetailsDriftDetectionFragment_workspace" | "WorkspaceDetailsEmptyFragment_workspace" | "WorkspaceDetailsStatusPanelFragment_workspace" | "WorkspaceNotificationPreferenceFragment_workspace">;
  readonly " $fragmentType": "WorkspaceDetailsIndexFragment_workspace";
};
export type WorkspaceDetailsIndexFragment_workspace$key = {
  readonly " $data"?: WorkspaceDetailsIndexFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceDetailsIndexFragment_workspace">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceDetailsIndexFragment_workspace",
  "selections": [
    (v0/*: any*/),
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
      "kind": "ScalarField",
      "name": "locked",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "destroyed",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "preventDestroyPlan",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "WorkspaceLabel",
      "kind": "LinkedField",
      "name": "labels",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "key",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "value",
          "storageKey": null
        }
      ],
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
      "concreteType": "WorkspaceAssessment",
      "kind": "LinkedField",
      "name": "assessment",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "hasDrift",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceDetailsEmptyFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceDetailsStatusPanelFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceNotificationPreferenceFragment_workspace"
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Run",
      "kind": "LinkedField",
      "name": "currentApplyRun",
      "plural": false,
      "selections": [
        (v0/*: any*/)
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "StateVersion",
      "kind": "LinkedField",
      "name": "currentStateVersion",
      "plural": false,
      "selections": [
        (v0/*: any*/),
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "StateVersionOutputsFragment_outputs"
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "StateVersionInventory",
          "kind": "LinkedField",
          "name": "inventory",
          "plural": false,
          "selections": [
            {
              "args": null,
              "kind": "FragmentSpread",
              "name": "StateVersionResourcesFragment_resources"
            },
            {
              "args": null,
              "kind": "FragmentSpread",
              "name": "StateVersionDependenciesFragment_dependencies"
            },
            {
              "args": null,
              "kind": "FragmentSpread",
              "name": "StateVersionCheckResultsFragment_checkResults"
            }
          ],
          "storageKey": null
        },
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "StateVersionFileFragment_stateVersion"
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "Run",
          "kind": "LinkedField",
          "name": "run",
          "plural": false,
          "selections": [
            (v0/*: any*/),
            {
              "args": null,
              "kind": "FragmentSpread",
              "name": "StateVersionInputVariablesFragment_variables"
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceDetailsDriftDetectionFragment_workspace"
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};
})();

(node as any).hash = "5e9d634cdbbb73081aa413fbdb06a4d6";

export default node;
