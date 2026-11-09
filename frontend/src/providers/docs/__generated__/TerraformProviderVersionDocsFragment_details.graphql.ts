/**
 * @generated SignedSource<<577db8c4a786a416b4734ae604a2b6b4>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
export type TerraformProviderDocItemCategory = "DATA_SOURCES" | "EPHEMERAL_RESOURCES" | "FUNCTIONS" | "GUIDES" | "OVERVIEW" | "RESOURCES" | "%future added value";
export type TerraformProviderVersionDocsProcessingStatus = "COMPLETED" | "FAILED" | "NOT_STARTED" | "PENDING" | "%future added value";
import { FragmentRefs } from "relay-runtime";
export type TerraformProviderVersionDocsFragment_details$data = {
  readonly docs: {
    readonly items: ReadonlyArray<{
      readonly category: TerraformProviderDocItemCategory;
      readonly filename: string;
    }>;
    readonly processingErrorMessage: string | null | undefined;
    readonly processingStatus: TerraformProviderVersionDocsProcessingStatus;
  };
  readonly id: string;
  readonly readme: string;
  readonly readmeUploaded: boolean;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformProviderVersionDocsSidebarFragment_details">;
  readonly " $fragmentType": "TerraformProviderVersionDocsFragment_details";
};
export type TerraformProviderVersionDocsFragment_details$key = {
  readonly " $data"?: TerraformProviderVersionDocsFragment_details$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformProviderVersionDocsFragment_details">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "TerraformProviderVersionDocsFragment_details",
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
      "name": "readme",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "readmeUploaded",
      "storageKey": null
    },
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
          "kind": "ScalarField",
          "name": "processingStatus",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "processingErrorMessage",
          "storageKey": null
        },
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
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "TerraformProviderVersionDocsSidebarFragment_details"
    }
  ],
  "type": "TerraformProviderVersion",
  "abstractKey": null
};

(node as any).hash = "2bb05389c6e8997c323fb4ca9a5c2068";

export default node;
