/**
 * @generated SignedSource<<f02bcdf55d7ee82906a2add079e900bf>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type TerraformProviderDocItemCategory = "DATA_SOURCES" | "EPHEMERAL_RESOURCES" | "FUNCTIONS" | "GUIDES" | "OVERVIEW" | "RESOURCES" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type TerraformProviderVersionDocsSidebarFragment_details$data = {
  readonly docs: {
    readonly items: ReadonlyArray<{
      readonly category: TerraformProviderDocItemCategory;
      readonly filename: string;
      readonly subcategory: string | null | undefined;
    }>;
  };
  readonly " $fragmentType": "TerraformProviderVersionDocsSidebarFragment_details";
};
export type TerraformProviderVersionDocsSidebarFragment_details$key = {
  readonly " $data"?: TerraformProviderVersionDocsSidebarFragment_details$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformProviderVersionDocsSidebarFragment_details">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformProviderVersionDocsSidebarFragment_details",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformProviderVersionDocs",
      "kind": "LinkedField",
      "name": "docs",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "TerraformProviderDocItem",
          "kind": "LinkedField",
          "name": "items",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "filename",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "category",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "subcategory",
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "TerraformProviderVersion",
  "abstractKey": null
};

(node as any).hash = "a30315f561ecd718f1713fdfe57c7eb4";

export default node;
