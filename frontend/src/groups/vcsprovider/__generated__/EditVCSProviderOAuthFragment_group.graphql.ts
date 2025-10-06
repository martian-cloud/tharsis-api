/**
 * @generated SignedSource<<5710982978ef07252043512a527db582>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { Fragment, ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type EditVCSProviderOAuthFragment_group$data = {
  readonly fullPath: string;
  readonly id: string;
  readonly " $fragmentType": "EditVCSProviderOAuthFragment_group";
};
export type EditVCSProviderOAuthFragment_group$key = {
  readonly " $data"?: EditVCSProviderOAuthFragment_group$data;
  readonly " $fragmentSpreads": FragmentRefs<"EditVCSProviderOAuthFragment_group">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "EditVCSProviderOAuthFragment_group",
  "selections": [
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "id",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "fullPath",
      "storageKey": null
    }
  ],
  "type": "Group",
  "abstractKey": null
};

(node as any).hash = "2a382dd62c19351e3574a3b06536751c";

export default node;
