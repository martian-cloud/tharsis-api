/**
 * @generated SignedSource<<c2d3bb0c2931437736535a1de4da40bc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type ActivityEventTargetType = "FEDERATED_REGISTRY" | "GPG_KEY" | "GROUP" | "MANAGED_IDENTITY" | "MANAGED_IDENTITY_ACCESS_RULE" | "NAMESPACE_MEMBERSHIP" | "PACKAGE" | "PACKAGE_VERSION" | "POLICY" | "ROLE" | "RUN" | "RUNNER" | "RUN_GATE" | "SERVICE_ACCOUNT" | "STATE_VERSION" | "TEAM" | "TEAM_MEMBER" | "TERRAFORM_MODULE" | "TERRAFORM_MODULE_VERSION" | "TERRAFORM_PROVIDER" | "TERRAFORM_PROVIDER_VERSION" | "TERRAFORM_PROVIDER_VERSION_MIRROR" | "VARIABLE" | "VCS_PROVIDER" | "WORKSPACE" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type ActivityEventTargetNotFoundFragment_event$data = {
  readonly targetType: ActivityEventTargetType;
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventListItemFragment_event">;
  readonly " $fragmentType": "ActivityEventTargetNotFoundFragment_event";
};
export type ActivityEventTargetNotFoundFragment_event$key = {
  readonly " $data"?: ActivityEventTargetNotFoundFragment_event$data;
  readonly " $fragmentSpreads": FragmentRefs<"ActivityEventTargetNotFoundFragment_event">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "ActivityEventTargetNotFoundFragment_event",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "targetType",
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "ActivityEventListItemFragment_event"
    }
  ],
  "type": "ActivityEvent",
  "abstractKey": null
};

(node as any).hash = "a6c11430d44b338ebb8441178c038190";

export default node;
