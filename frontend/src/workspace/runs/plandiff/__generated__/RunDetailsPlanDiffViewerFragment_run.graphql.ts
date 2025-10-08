/**
 * @generated SignedSource<<516b9b67ed0aee20ccc78558b2811331>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type PlanChangeAction = "CREATE" | "CREATE_THEN_DELETE" | "DELETE" | "DELETE_THEN_CREATE" | "NOOP" | "READ" | "UPDATE" | "%future added value";
export type PlanChangeWarningType = "after" | "before" | "%future added value";
export type PlanStatus = "canceled" | "errored" | "finished" | "pending" | "queued" | "running" | "%future added value";
export type TerraformResourceMode = "data" | "managed" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type RunDetailsPlanDiffViewerFragment_run$data = {
  readonly plan: {
    readonly changes: {
      readonly outputs: ReadonlyArray<{
        readonly action: PlanChangeAction;
        readonly originalSource: string;
        readonly outputName: string;
        readonly unifiedDiff: string;
        readonly warnings: ReadonlyArray<{
          readonly changeType: PlanChangeWarningType;
          readonly line: number;
          readonly message: string;
        }>;
      }>;
      readonly resources: ReadonlyArray<{
        readonly action: PlanChangeAction;
        readonly address: string;
        readonly drifted: boolean;
        readonly imported: boolean;
        readonly mode: TerraformResourceMode;
        readonly moduleAddress: string;
        readonly originalSource: string;
        readonly providerName: string;
        readonly resourceName: string;
        readonly resourceType: string;
        readonly unifiedDiff: string;
        readonly warnings: ReadonlyArray<{
          readonly changeType: PlanChangeWarningType;
          readonly line: number;
          readonly message: string;
        }>;
      }>;
    } | null | undefined;
    readonly status: PlanStatus;
  };
  readonly " $fragmentType": "RunDetailsPlanDiffViewerFragment_run";
};
export type RunDetailsPlanDiffViewerFragment_run$key = {
  readonly " $data"?: RunDetailsPlanDiffViewerFragment_run$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunDetailsPlanDiffViewerFragment_run">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "action",
  "storageKey": null
},
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "unifiedDiff",
  "storageKey": null
},
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "originalSource",
  "storageKey": null
},
v3 = {
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
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunDetailsPlanDiffViewerFragment_run",
  "selections": [
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
          "kind": "ScalarField",
          "name": "status",
          "storageKey": null
        },
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
                (v0/*: any*/),
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
                  "name": "providerName",
                  "storageKey": null
                },
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "resourceType",
                  "storageKey": null
                },
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "resourceName",
                  "storageKey": null
                },
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "moduleAddress",
                  "storageKey": null
                },
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "mode",
                  "storageKey": null
                },
                (v1/*: any*/),
                (v2/*: any*/),
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
                  "name": "imported",
                  "storageKey": null
                },
                (v3/*: any*/)
              ],
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "concreteType": "PlanOutputChange",
              "kind": "LinkedField",
              "name": "outputs",
              "plural": true,
              "selections": [
                (v0/*: any*/),
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "outputName",
                  "storageKey": null
                },
                (v1/*: any*/),
                (v2/*: any*/),
                (v3/*: any*/)
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
  "type": "Run",
  "abstractKey": null
};
})();

(node as any).hash = "f2cd33061ea87f2358297da3639800f5";

export default node;
