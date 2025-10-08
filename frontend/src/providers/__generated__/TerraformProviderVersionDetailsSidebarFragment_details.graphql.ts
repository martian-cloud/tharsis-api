/**
 * @generated SignedSource<<4cd1929cb9a2cfb9a6f21205e510e6e4>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformProviderVersionDetailsSidebarFragment_details$data = {
  readonly createdBy: string;
  readonly gpgKeyId: string | null | undefined;
  readonly latest: boolean;
  readonly metadata: {
    readonly createdAt: any;
  };
  readonly platforms: ReadonlyArray<{
    readonly arch: string;
    readonly binaryUploaded: boolean;
    readonly id: string;
    readonly os: string;
  }>;
  readonly protocols: ReadonlyArray<string>;
  readonly provider: {
    readonly id: string;
    readonly name: string;
    readonly private: boolean;
    readonly registryNamespace: string;
    readonly repositoryUrl: string;
  };
  readonly version: string;
  readonly " $fragmentType": "TerraformProviderVersionDetailsSidebarFragment_details";
};
export type TerraformProviderVersionDetailsSidebarFragment_details$key = {
  readonly " $data"?: TerraformProviderVersionDetailsSidebarFragment_details$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformProviderVersionDetailsSidebarFragment_details">;
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
  "name": "TerraformProviderVersionDetailsSidebarFragment_details",
  "selections": [
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
      "name": "createdBy",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "gpgKeyId",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "protocols",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "kind": "ScalarField",
      "name": "latest",
      "storageKey": null
    },
    {
      "alias": null,
      "args": null,
      "concreteType": "TerraformProviderPlatform",
      "kind": "LinkedField",
      "name": "platforms",
      "plural": true,
      "selections": [
        (v0/*: any*/),
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "os",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "arch",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "binaryUploaded",
          "storageKey": null
        }
      ],
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
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "repositoryUrl",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "TerraformProviderVersion",
  "abstractKey": null
};
})();

(node as any).hash = "3fe4bf09bf7ac0d508943f6e8e7d1ebb";

export default node;
