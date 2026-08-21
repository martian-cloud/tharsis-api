/**
 * @generated SignedSource<<498425ddec7606e5332f4ff86d5d4025>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type PolicyCheckStatus = "CANCELED" | "CREATED" | "ERRORED" | "OVERRIDDEN" | "PASSED" | "PENDING" | "QUEUED" | "RUNNING" | "SKIPPED" | "SOFT_FAILED" | "%future added value";
export type RunTaskStageName = "POST_APPLY" | "POST_PLAN" | "PRE_APPLY" | "PRE_PLAN" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type ApprovalGateCardFragment_gate$data = {
  readonly id: string;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly policyCheck: {
    readonly checkType: string;
    readonly id: string;
    readonly messagesSummary: {
      readonly messages: ReadonlyArray<string>;
      readonly truncated: boolean;
    };
    readonly policies: ReadonlyArray<{
      readonly id: string;
      readonly status: string;
    }>;
    readonly stageName: RunTaskStageName;
    readonly status: PolicyCheckStatus;
  };
  readonly run: {
    readonly id: string;
    readonly isDestroy: boolean;
    readonly workspace: {
      readonly fullPath: string;
    };
  };
  readonly " $fragmentType": "ApprovalGateCardFragment_gate";
};
export type ApprovalGateCardFragment_gate$key = {
  readonly " $data"?: ApprovalGateCardFragment_gate$data;
  readonly " $fragmentSpreads": FragmentRefs<"ApprovalGateCardFragment_gate">;
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
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ApprovalGateCardFragment_gate",
  "selections": [
    (v0/*: any*/),
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
          "name": "createdAt",
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
        (v0/*: any*/),
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
          "concreteType": "Workspace",
          "kind": "LinkedField",
          "name": "workspace",
          "plural": false,
          "selections": [
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "fullPath",
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
      "concreteType": "PolicyCheck",
      "kind": "LinkedField",
      "name": "policyCheck",
      "plural": false,
      "selections": [
        (v0/*: any*/),
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "checkType",
          "storageKey": null
        },
        (v1/*: any*/),
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "stageName",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "PolicyCheckPolicy",
          "kind": "LinkedField",
          "name": "policies",
          "plural": true,
          "selections": [
            (v0/*: any*/),
            (v1/*: any*/)
          ],
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "PolicyCheckMessagesSummary",
          "kind": "LinkedField",
          "name": "messagesSummary",
          "plural": false,
          "selections": [
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "messages",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "truncated",
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "RunGate",
  "abstractKey": null
};
})();

(node as any).hash = "e9f8ac870c425c3e08bc75181cd7f70a";

export default node;
