/**
 * @generated SignedSource<<ed6d3bfa7ee95044cba534ba8183ce5c>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ApplyStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "%future added value";
export type PlanStatus = "canceled" | "errored" | "finished" | "pending" | "queued" | "running" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "applying" | "canceled" | "errored" | "pending" | "plan_queued" | "planned" | "planned_and_finished" | "planning" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type WorkspaceDetailsIndexFragment_workspace$data = {
  readonly assessment: {
    readonly hasDrift: boolean;
  } | null | undefined;
  readonly currentJob: {
    readonly id: string;
  } | null | undefined;
  readonly currentStateVersion: {
    readonly id: string;
    readonly metadata: {
      readonly createdAt: any;
    };
    readonly run: {
      readonly apply: {
        readonly metadata: {
          readonly createdAt: any;
          readonly updatedAt: any;
        };
        readonly status: ApplyStatus;
        readonly triggeredBy: string | null | undefined;
      } | null | undefined;
      readonly configurationVersion: {
        readonly id: string;
        readonly vcsEvent: {
          readonly status: string;
        } | null | undefined;
      } | null | undefined;
      readonly createdBy: string;
      readonly id: string;
      readonly isDestroy: boolean;
      readonly metadata: {
        readonly createdAt: any;
      };
      readonly moduleSource: string | null | undefined;
      readonly moduleVersion: string | null | undefined;
      readonly plan: {
        readonly metadata: {
          readonly createdAt: any;
        };
        readonly status: PlanStatus;
      };
      readonly status: RunStatus;
      readonly " $fragmentSpreads": FragmentRefs<"StateVersionInputVariablesFragment_variables">;
    } | null | undefined;
    readonly " $fragmentSpreads": FragmentRefs<"StateVersionDependenciesFragment_dependencies" | "StateVersionFileFragment_stateVersion" | "StateVersionOutputsFragment_outputs" | "StateVersionResourcesFragment_resources">;
  } | null | undefined;
  readonly description: string;
  readonly fullPath: string;
  readonly id: string;
  readonly metadata: {
    readonly trn: string;
  };
  readonly name: string;
  readonly preventDestroyPlan: boolean;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceDetailsCurrentJobFragment_workspace" | "WorkspaceDetailsDriftDetectionFragment_workspace" | "WorkspaceDetailsEmptyFragment_workspace" | "WorkspaceNotificationPreferenceFragment_workspace">;
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
},
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdAt",
  "storageKey": null
},
v2 = {
  "alias": null,
  "args": null,
  "concreteType": "ResourceMetadata",
  "kind": "LinkedField",
  "name": "metadata",
  "plural": false,
  "selections": [
    (v1/*: any*/)
  ],
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
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
      "name": "preventDestroyPlan",
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
      "name": "WorkspaceDetailsCurrentJobFragment_workspace"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "WorkspaceNotificationPreferenceFragment_workspace"
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Job",
      "kind": "LinkedField",
      "name": "currentJob",
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
          "name": "StateVersionFileFragment_stateVersion"
        },
        (v2/*: any*/),
        {
          "alias": null,
          "args": null,
          "concreteType": "Run",
          "kind": "LinkedField",
          "name": "run",
          "plural": false,
          "selections": [
            {
              "args": null,
              "kind": "FragmentSpread",
              "name": "StateVersionInputVariablesFragment_variables"
            },
            (v0/*: any*/),
            (v3/*: any*/),
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "createdBy",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "isDestroy",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "moduleSource",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "moduleVersion",
              "storageKey": null
            },
            (v2/*: any*/),
            {
              "alias": null,
              "args": null,
              "concreteType": "ConfigurationVersion",
              "kind": "LinkedField",
              "name": "configurationVersion",
              "plural": false,
              "selections": [
                (v0/*: any*/),
                {
                  "alias": null,
                  "args": null,
                  "concreteType": "VCSEvent",
                  "kind": "LinkedField",
                  "name": "vcsEvent",
                  "plural": false,
                  "selections": [
                    (v3/*: any*/)
                  ],
                  "storageKey": null
                }
              ],
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "concreteType": "Plan",
              "kind": "LinkedField",
              "name": "plan",
              "plural": false,
              "selections": [
                (v3/*: any*/),
                (v2/*: any*/)
              ],
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "concreteType": "Apply",
              "kind": "LinkedField",
              "name": "apply",
              "plural": false,
              "selections": [
                (v3/*: any*/),
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "triggeredBy",
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
                    (v1/*: any*/),
                    {
                      "alias": null,
                      "args": null,
                      "kind": "ScalarField",
                      "name": "updatedAt",
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

(node as any).hash = "5d246777a06694813b1238c8eebda652";

export default node;
