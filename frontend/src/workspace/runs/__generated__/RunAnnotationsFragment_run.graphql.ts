/**
 * @generated SignedSource<<2c79d11ac3703f482f816f2b274c38cb>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type RunAnnotationsFragment_run$data = {
  readonly annotations: ReadonlyArray<{
    readonly key: string;
    readonly link: string | null | undefined;
    readonly value: string;
  }>;
  readonly " $fragmentType": "RunAnnotationsFragment_run";
};
export type RunAnnotationsFragment_run$key = {
  readonly " $data"?: RunAnnotationsFragment_run$data;
  readonly " $fragmentSpreads": FragmentRefs<"RunAnnotationsFragment_run">;
};

const node: ReaderFragment = {
  "argumentDefinitions": [],
  "kind": "Fragment",
  "metadata": null,
  "name": "RunAnnotationsFragment_run",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "RunAnnotation",
      "kind": "LinkedField",
      "name": "annotations",
      "plural": true,
      "selections": [
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "key",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "value",
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "kind": "ScalarField",
          "name": "link",
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Run",
  "abstractKey": null
};

(node as any).hash = "8aa0d3bc1ca803c158971db5498c78ac";

export default node;
