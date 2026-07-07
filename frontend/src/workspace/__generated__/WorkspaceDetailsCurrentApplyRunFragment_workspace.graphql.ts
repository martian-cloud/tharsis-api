/**
 * @generated SignedSource<<0f19d66664a1c7b2fb2749810237c454>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ApplyStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "skipped" | "%future added value";
export type PlanStatus = "canceled" | "created" | "errored" | "finished" | "pending" | "queued" | "running" | "%future added value";
export type RunStatus = "applied" | "apply_queued" | "applying" | "canceled" | "discarded" | "errored" | "pending" | "plan_queued" | "planned" | "planned_and_finished" | "planning" | "queuing" | "queuing_apply" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type WorkspaceDetailsCurrentApplyRunFragment_workspace$data = {
  readonly currentApplyRun: {
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
  } | null | undefined;
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "WorkspaceDetailsCurrentApplyRunFragment_workspace";
};
export type WorkspaceDetailsCurrentApplyRunFragment_workspace$key = {
  readonly " $data"?: WorkspaceDetailsCurrentApplyRunFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceDetailsCurrentApplyRunFragment_workspace">;
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
  "name": "status",
  "storageKey": null
},
v2 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "createdAt",
  "storageKey": null
},
v3 = {
  "alias": null,
  "args": null,
  "concreteType": "ResourceMetadata",
  "kind": "LinkedField",
  "name": "metadata",
  "plural": false,
  "selections": [
    (v2/*: any*/)
  ],
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "WorkspaceDetailsCurrentApplyRunFragment_workspace",
  "selections": [
    (v0/*: any*/),
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
        (v3/*: any*/),
        {
          "alias": null,
          "args": null,
          "concreteType": "ConfigurationVersion",
          "kind": "LinkedField",
          "name": "configurationVersion",
          "plural": false,
          "selections": [
            (v0/*: any*/)
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
            (v1/*: any*/),
            (v3/*: any*/)
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
            (v1/*: any*/),
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
                (v2/*: any*/),
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
  "type": "Workspace",
  "abstractKey": null
};
})();

(node as any).hash = "8b5a17bdb5c542c2683ab925215dcc25";

export default node;
