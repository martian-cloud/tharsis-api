/**
 * @generated SignedSource<<c7c4f1a906fc5c7e53949756eb4b6b84>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type MaxJobDurationSettingFragment_workspace$data = {
  readonly maxJobDuration: number;
  readonly " $fragmentType": "MaxJobDurationSettingFragment_workspace";
};
export type MaxJobDurationSettingFragment_workspace$key = {
  readonly " $data"?: MaxJobDurationSettingFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"MaxJobDurationSettingFragment_workspace">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "MaxJobDurationSettingFragment_workspace",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "maxJobDuration",
      "storageKey": null
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};

(node as any).hash = "024d03edfa7416a0472e80efe6224cf4";

export default node;
