/**
 * @generated SignedSource<<8aca5cfab3c9eb76c5ac87cf83b6c62a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type PolicyCheckStatus = "CANCELED" | "CREATED" | "ERRORED" | "OVERRIDDEN" | "PASSED" | "PENDING" | "QUEUED" | "RUNNING" | "SKIPPED" | "SOFT_FAILED" | "%future added value";
export type PolicyEnforcementLevel = "ADVISORY" | "HARD_MANDATORY" | "SOFT_MANDATORY" | "%future added value";
export type RunGateDecision = "APPROVE" | "REJECT" | "%future added value";
export type RunGateStatus = "APPROVED" | "CANCELED" | "OVERRIDDEN" | "PENDING" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type RunTaskStagePolicyCheckPanelFragment_check$data = {
  readonly checkType: string;
  readonly currentJob: {
    readonly id: string;
    readonly timestamps: {
      readonly finishedAt: any | null | undefined;
      readonly runningAt: any | null | undefined;
    };
  } | null | undefined;
  readonly jobs: {
    readonly totalCount: number;
  };
  readonly messagesSummary: {
    readonly messages: ReadonlyArray<string>;
    readonly truncated: boolean;
  };
  readonly nodePath: string;
  readonly policies: ReadonlyArray<{
    readonly enforcementLevel: PolicyEnforcementLevel;
    readonly id: string;
    readonly status: string;
  }>;
  readonly runGate: {
    readonly approvalRules: ReadonlyArray<{
      readonly name: string;
      readonly requiredApprovals: number;
    }>;
    readonly approvals: ReadonlyArray<{
      readonly comment: string;
      readonly coveredRules: ReadonlyArray<string>;
      readonly createdBy: string;
      readonly decision: RunGateDecision;
      readonly id: string;
      readonly metadata: {
        readonly createdAt: any;
      };
      readonly serviceAccount: {
        readonly id: string;
        readonly name: string;
        readonly resourcePath: string;
      } | null | undefined;
      readonly user: {
        readonly email: string;
        readonly id: string;
        readonly username: string;
      } | null | undefined;
    }>;
    readonly id: string;
    readonly metadata: {
      readonly updatedAt: any;
    };
    readonly overriddenBy: string | null | undefined;
    readonly overrideComment: string;
    readonly status: RunGateStatus;
  } | null | undefined;
  readonly status: PolicyCheckStatus;
  readonly " $fragmentType": "RunTaskStagePolicyCheckPanelFragment_check";
};
export type RunTaskStagePolicyCheckPanelFragment_check$key = {
  readonly " $data"?: RunTaskStagePolicyCheckPanelFragment_check$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunTaskStagePolicyCheckPanelFragment_check">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "status",
  "storageKey": null
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
  "kind": "ScalarField",
  "name": "name",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunTaskStagePolicyCheckPanelFragment_check",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "checkType",
      "storageKey": null
    },
    (v0/*: any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "nodePath",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "Job",
      "kind": "LinkedField",
      "name": "currentJob",
      "plural": false,
      "selections": [
        (v1/*: any*/),
        {
          "alias": null,
          "args": null,
          "concreteType": "JobTimestamps",
          "kind": "LinkedField",
          "name": "timestamps",
          "plural": false,
          "selections": [
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "runningAt",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "finishedAt",
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
      "args": [
        {
          "kind": "Literal",
          "name": "first",
          "value": 0
        }
      ],
      "concreteType": "JobConnection",
      "kind": "LinkedField",
      "name": "jobs",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "totalCount",
          "storageKey": null
        }
      ],
      "storageKey": "jobs(first:0)"
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "PolicyCheckPolicy",
      "kind": "LinkedField",
      "name": "policies",
      "plural": true,
      "selections": [
        (v1/*: any*/),
        (v0/*: any*/),
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "enforcementLevel",
          "storageKey": null
        }
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
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "RunGate",
      "kind": "LinkedField",
      "name": "runGate",
      "plural": false,
      "selections": [
        (v1/*: any*/),
        (v0/*: any*/),
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "overriddenBy",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "overrideComment",
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
              "name": "updatedAt",
              "storageKey": null
            }
          ],
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "RunGateApprovalRule",
          "kind": "LinkedField",
          "name": "approvalRules",
          "plural": true,
          "selections": [
            (v2/*: any*/),
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "requiredApprovals",
              "storageKey": null
            }
          ],
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "RunGateApproval",
          "kind": "LinkedField",
          "name": "approvals",
          "plural": true,
          "selections": [
            (v1/*: any*/),
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "decision",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "comment",
              "storageKey": null
            },
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
              "name": "coveredRules",
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
                  "name": "createdAt",
                  "storageKey": null
                }
              ],
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "concreteType": "User",
              "kind": "LinkedField",
              "name": "user",
              "plural": false,
              "selections": [
                (v1/*: any*/),
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "username",
                  "storageKey": null
                },
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "email",
                  "storageKey": null
                }
              ],
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "concreteType": "ServiceAccount",
              "kind": "LinkedField",
              "name": "serviceAccount",
              "plural": false,
              "selections": [
                (v1/*: any*/),
                (v2/*: any*/),
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "resourcePath",
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
  "type": "PolicyCheck",
  "abstractKey": null
};
})();

(node as any).hash = "9b0e25b37219b74402399f89d6b9003c";

export default node;
