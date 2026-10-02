/**
 * @generated SignedSource<<d26e16a2ee1535783a1489d8d8d9eae3>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ApplyStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "skipped" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "apply_queuing" | "applying" | "canceled" | "discarded" | "errored" | "pending" | "plan_queued" | "plan_queuing" | "planned" | "planned_and_finished" | "planning" | "post_apply_running" | "post_plan_awaiting_decision" | "post_plan_running" | "pre_apply_awaiting_decision" | "pre_apply_completed" | "pre_apply_queuing" | "pre_apply_running" | "pre_plan_awaiting_decision" | "pre_plan_completed" | "pre_plan_queuing" | "pre_plan_running" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type WorkspaceDetailsStatusPanelFragment_workspace$data = {
  readonly currentApplyRun: {
    readonly id: string;
    readonly isDestroy: boolean;
    readonly " $fragmentSpreads": FragmentRefs<"RunStageIconsFragment_run">;
  } | null | undefined;
  readonly currentStateVersion: {
    readonly id: string;
    readonly inventory: {
      readonly resources: ReadonlyArray<{
        readonly __typename: "StateVersionResource";
      }>;
    };
    readonly metadata: {
      readonly createdAt: any;
    };
    readonly run: {
      readonly apply: {
        readonly status: ApplyStatus;
      } | null | undefined;
      readonly configurationVersion: {
        readonly id: string;
        readonly metadata: {
          readonly createdAt: any;
        };
        readonly vcsEvent: {
          readonly id: string;
        } | null | undefined;
      } | null | undefined;
      readonly id: string;
      readonly moduleSource: string | null | undefined;
      readonly moduleVersion: string | null | undefined;
      readonly plan: {
        readonly summary: {
          readonly resourceAdditions: number;
          readonly resourceChanges: number;
          readonly resourceDestructions: number;
        };
      };
      readonly status: RunStatus;
      readonly " $fragmentSpreads": FragmentRefs<"ModuleSourceLinkFragment_run" | "RunStageIconsFragment_run">;
    } | null | undefined;
  } | null | undefined;
  readonly fullPath: string;
  readonly " $fragmentType": "WorkspaceDetailsStatusPanelFragment_workspace";
};
export type WorkspaceDetailsStatusPanelFragment_workspace$key = {
  readonly " $data"?: WorkspaceDetailsStatusPanelFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceDetailsStatusPanelFragment_workspace">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "args": null,
  "kind": "FragmentSpread",
  "name": "RunStageIconsFragment_run"
},
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
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
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "createdAt",
      "storageKey": null
    }
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
  "name": "WorkspaceDetailsStatusPanelFragment_workspace",
  "selections": [
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
      "concreteType": "Run",
      "kind": "LinkedField",
      "name": "currentApplyRun",
      "plural": false,
      "selections": [
        (v0/*: any*/),
        (v1/*: any*/),
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "isDestroy",
          "storageKey": null
        }
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
        (v1/*: any*/),
        (v2/*: any*/),
        {
          "alias": null,
          "args": null,
          "concreteType": "StateVersionInventory",
          "kind": "LinkedField",
          "name": "inventory",
          "plural": false,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "StateVersionResource",
              "kind": "LinkedField",
              "name": "resources",
              "plural": true,
              "selections": [
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "__typename",
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
          "args": null,
          "concreteType": "Run",
          "kind": "LinkedField",
          "name": "run",
          "plural": false,
          "selections": [
            {
              "args": null,
              "kind": "FragmentSpread",
              "name": "ModuleSourceLinkFragment_run"
            },
            (v0/*: any*/),
            (v1/*: any*/),
            (v3/*: any*/),
            {
              "alias": null,
              "args": null,
              "concreteType": "Apply",
              "kind": "LinkedField",
              "name": "apply",
              "plural": false,
              "selections": [
                (v3/*: any*/)
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
                {
                  "alias": null,
                  "args": null,
                  "concreteType": "PlanSummary",
                  "kind": "LinkedField",
                  "name": "summary",
                  "plural": false,
                  "selections": [
                    {
                      "alias": null,
                      "args": null,
                      "kind": "ScalarField",
                      "name": "resourceAdditions",
                      "storageKey": null
                    },
                    {
                      "alias": null,
                      "args": null,
                      "kind": "ScalarField",
                      "name": "resourceChanges",
                      "storageKey": null
                    },
                    {
                      "alias": null,
                      "args": null,
                      "kind": "ScalarField",
                      "name": "resourceDestructions",
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
            {
              "alias": null,
              "args": null,
              "concreteType": "ConfigurationVersion",
              "kind": "LinkedField",
              "name": "configurationVersion",
              "plural": false,
              "selections": [
                (v1/*: any*/),
                (v2/*: any*/),
                {
                  "alias": null,
                  "args": null,
                  "concreteType": "VCSEvent",
                  "kind": "LinkedField",
                  "name": "vcsEvent",
                  "plural": false,
                  "selections": [
                    (v1/*: any*/)
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
  ],
  "type": "Workspace",
  "abstractKey": null
};
})();

(node as any).hash = "99e02ab74f44a2d19638be0f63ab561d";

export default node;
