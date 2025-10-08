/**
 * @generated SignedSource<<1d53cc0fccf463d4582f12d05dd48643>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type TerraformModuleVersionDetailsIndexFragment_details$data = {
  readonly configurationDetails: {
    readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionDocsFragment_configurationDetails">;
  } | null | undefined;
  readonly id: string;
  readonly metadata: {
    readonly trn: string;
  };
  readonly module: {
    readonly id: string;
    readonly name: string;
    readonly private: boolean;
    readonly registryNamespace: string;
    readonly source: string;
    readonly system: string;
    readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionListFragment_module">;
  };
  readonly status: string;
  readonly version: string;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionAttestListFragment_attestations" | "TerraformModuleVersionDetailsSidebarFragment_details">;
  readonly " $fragmentType": "TerraformModuleVersionDetailsIndexFragment_details";
};
export type TerraformModuleVersionDetailsIndexFragment_details$key = {
  readonly " $data"?: TerraformModuleVersionDetailsIndexFragment_details$data;
  readonly " $fragmentSpreads": FragmentRefs<"TerraformModuleVersionDetailsIndexFragment_details">;
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
  "name": "TerraformModuleVersionDetailsIndexFragment_details",
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
      "name": "status",
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
      "concreteType": "TerraformModule",
      "kind": "LinkedField",
      "name": "module",
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
          "name": "source",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "system",
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
          "name": "TerraformModuleVersionListFragment_module"
        }
      ],
      "storageKey": null
    },
    {
      "alias": null,
      "args": [
        {
          "kind": "Literal",
          "name": "path",
          "value": "root"
        }
      ],
      "concreteType": "TerraformModuleConfigurationDetails",
      "kind": "LinkedField",
      "name": "configurationDetails",
      "plural": false,
      "selections": [
        {
          "args": null,
          "kind": "FragmentSpread",
          "name": "TerraformModuleVersionDocsFragment_configurationDetails"
        }
      ],
      "storageKey": "configurationDetails(path:\"root\")"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "TerraformModuleVersionAttestListFragment_attestations"
    },
    {
      "args": null,
      "kind": "FragmentSpread",
      "name": "TerraformModuleVersionDetailsSidebarFragment_details"
    }
  ],
  "type": "TerraformModuleVersion",
  "abstractKey": null
};
})();

(node as any).hash = "c4e8b6416cb733b294499a71614d1422";

export default node;
