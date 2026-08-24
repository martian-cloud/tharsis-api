/**
 * @generated SignedSource<<64215b15110c07372b4c4cbe05a51180>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type RunGateDecision = "APPROVE" | "REJECT" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type RunTaskStagePolicyApproversBoxFragment_gate$data = {
  readonly approvalRules: ReadonlyArray<{
    readonly allowedServiceAccounts: ReadonlyArray<{
      readonly id: string;
      readonly name: string;
      readonly resourcePath: string;
    }>;
    readonly allowedTeams: ReadonlyArray<{
      readonly id: string;
      readonly name: string;
    }>;
    readonly allowedUsers: ReadonlyArray<{
      readonly email: string;
      readonly id: string;
      readonly username: string;
    }>;
    readonly name: string;
    readonly requiredApprovals: number;
  }>;
  readonly approvals: ReadonlyArray<{
    readonly coveredRules: ReadonlyArray<string>;
    readonly createdBy: string;
    readonly decision: RunGateDecision;
    readonly id: string;
    readonly serviceAccount: {
      readonly id: string;
      readonly name: string;
      readonly resourcePath: string;
    } | null | undefined;
    readonly user: {
      readonly email: string;
      readonly id: string;
    } | null | undefined;
  }>;
  readonly " $fragmentType": "RunTaskStagePolicyApproversBoxFragment_gate";
};
export type RunTaskStagePolicyApproversBoxFragment_gate$key = {
  readonly " $data"?: RunTaskStagePolicyApproversBoxFragment_gate$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunTaskStagePolicyApproversBoxFragment_gate">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "name",
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
  "name": "email",
  "storageKey": null
},
v3 = [
  (v1/*: any*/),
  (v0/*: any*/),
  {
    "alias": null,
    "args": null,
    "kind": "ScalarField",
    "name": "resourcePath",
    "storageKey": null
  }
];
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunTaskStagePolicyApproversBoxFragment_gate",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "RunGateApprovalRule",
      "kind": "LinkedField",
      "name": "approvalRules",
      "plural": true,
      "selections": [
        (v0/*: any*/),
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "requiredApprovals",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "User",
          "kind": "LinkedField",
          "name": "allowedUsers",
          "plural": true,
          "selections": [
            (v1/*: any*/),
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "username",
              "storageKey": null
            },
            (v2/*: any*/)
          ],
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "ServiceAccount",
          "kind": "LinkedField",
          "name": "allowedServiceAccounts",
          "plural": true,
          "selections": (v3/*: any*/),
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "Team",
          "kind": "LinkedField",
          "name": "allowedTeams",
          "plural": true,
          "selections": [
            (v1/*: any*/),
            (v0/*: any*/)
          ],
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
          "concreteType": "User",
          "kind": "LinkedField",
          "name": "user",
          "plural": false,
          "selections": [
            (v1/*: any*/),
            (v2/*: any*/)
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
          "selections": (v3/*: any*/),
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

(node as any).hash = "654e91a998fb4a6c1c764df9abad83c6";

export default node;
