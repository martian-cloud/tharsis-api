/**
 * @generated SignedSource<<5c03beb7528a151e4242de83a3513223>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type PlanChangeAction = "CREATE" | "CREATE_THEN_DELETE" | "DELETE" | "DELETE_THEN_CREATE" | "NOOP" | "READ" | "UPDATE" | "%future added value";
export type PlanChangeWarningType = "after" | "before" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "applying" | "canceled" | "errored" | "pending" | "plan_queued" | "planned" | "planned_and_finished" | "planning" | "%future added value";
export type TerraformResourceMode = "data" | "managed" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type WorkspaceDetailsDriftViewerFragment_workspace$data = {
  readonly assessment: {
    readonly completedAt: any | null | undefined;
    readonly hasDrift: boolean;
    readonly run: {
      readonly plan: {
        readonly changes: {
          readonly resources: ReadonlyArray<{
            readonly action: PlanChangeAction;
            readonly address: string;
            readonly drifted: boolean;
            readonly imported: boolean;
            readonly mode: TerraformResourceMode;
            readonly originalSource: string;
            readonly unifiedDiff: string;
            readonly warnings: ReadonlyArray<{
              readonly changeType: PlanChangeWarningType;
              readonly line: number;
              readonly message: string;
            }>;
          }>;
        } | null | undefined;
      };
      readonly status: RunStatus;
    } | null | undefined;
    readonly startedAt: any;
  } | null | undefined;
  readonly " $fragmentType": "WorkspaceDetailsDriftViewerFragment_workspace";
};
export type WorkspaceDetailsDriftViewerFragment_workspace$key = {
  readonly " $data"?: WorkspaceDetailsDriftViewerFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceDetailsDriftViewerFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceDetailsDriftViewerFragment_workspace",
  "selections": [
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
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "startedAt",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "completedAt",
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
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "status",
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
                  "concreteType": "PlanChanges",
                  "kind": "LinkedField",
                  "name": "changes",
                  "plural": false,
                  "selections": [
                    {
                      "alias": null,
                      "args": null,
                      "concreteType": "PlanResourceChange",
                      "kind": "LinkedField",
                      "name": "resources",
                      "plural": true,
                      "selections": [
                        {
                          "alias": null,
                          "args": null,
                          "kind": "ScalarField",
                          "name": "action",
                          "storageKey": null
                        },
                        {
                          "alias": null,
                          "args": null,
                          "kind": "ScalarField",
                          "name": "originalSource",
                          "storageKey": null
                        },
                        {
                          "alias": null,
                          "args": null,
                          "kind": "ScalarField",
                          "name": "mode",
                          "storageKey": null
                        },
                        {
                          "alias": null,
                          "args": null,
                          "kind": "ScalarField",
                          "name": "address",
                          "storageKey": null
                        },
                        {
                          "alias": null,
                          "args": null,
                          "kind": "ScalarField",
                          "name": "imported",
                          "storageKey": null
                        },
                        {
                          "alias": null,
                          "args": null,
                          "kind": "ScalarField",
                          "name": "drifted",
                          "storageKey": null
                        },
                        {
                          "alias": null,
                          "args": null,
                          "kind": "ScalarField",
                          "name": "unifiedDiff",
                          "storageKey": null
                        },
                        {
                          "alias": null,
                          "args": null,
                          "concreteType": "PlanChangeWarning",
                          "kind": "LinkedField",
                          "name": "warnings",
                          "plural": true,
                          "selections": [
                            {
                              "alias": null,
                              "args": null,
                              "kind": "ScalarField",
                              "name": "line",
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
                              "name": "changeType",
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

(node as any).hash = "31d7e2431ac0bbd744ea5bc5b917bc23";

export default node;
