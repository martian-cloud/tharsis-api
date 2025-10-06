/**
 * @generated SignedSource<<8947fbac6f9a636ba1bc6e43317eb95f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformProviderVersionDetailsIndexFragment_details$data = {
  readonly id: string;
  readonly metadata: {
    readonly trn: string;
  };
  readonly provider: {
    readonly id: string;
    readonly name: string;
    readonly private: boolean;
    readonly registryNamespace: string;
    readonly " $fragmentSpreads": FragmentRefs<"TerraformProviderVersionListFragment_provider">;
  };
  readonly readme: string;
  readonly shaSumsSigUploaded: boolean;
  readonly shaSumsUploaded: boolean;
  readonly version: string;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformProviderVersionDetailsSidebarFragment_details">;
  readonly " $fragmentType": "TerraformProviderVersionDetailsIndexFragment_details";
};
export type TerraformProviderVersionDetailsIndexFragment_details$key = {
  readonly " $data"?: TerraformProviderVersionDetailsIndexFragment_details$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformProviderVersionDetailsIndexFragment_details">;
};

const node: ReaderFragment = (function(){
var v0 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformProviderVersionDetailsIndexFragment_details",
  "selections": [
    (v0/*: any*/),
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "version",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "readme",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "shaSumsUploaded",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "shaSumsSigUploaded",
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
          "name": "trn",
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformProvider",
      "kind": "LinkedField",
      "name": "provider",
      "plural": false,
      "selections": [
        (v0/*: any*/),
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "name",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "registryNamespace",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "private",
          "storageKey": null
        },
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "TerraformProviderVersionListFragment_provider"
        }
      ],
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "TerraformProviderVersionDetailsSidebarFragment_details"
    }
  ],
  "type": "TerraformProviderVersion",
  "abstractKey": null
};
})();

(node as any).hash = "badd79b185a63331ac41cfe2b70f9317";

export default node;
