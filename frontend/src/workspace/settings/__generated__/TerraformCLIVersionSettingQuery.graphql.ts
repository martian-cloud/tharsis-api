/**
 * @generated SignedSource<<c9a1ef0b95c649491fa5c7bca636bb1b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type TerraformCLIVersionSettingQuery$variables = Record<PropertyKey, never>;
export type TerraformCLIVersionSettingQuery$data = {
  readonly terraformCLIVersions: {
    readonly versions: ReadonlyArray<string>;
  } | null | undefined;
};
export type TerraformCLIVersionSettingQuery = {
  response: TerraformCLIVersionSettingQuery$data;
  variables: TerraformCLIVersionSettingQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "alias": null,
    "args": null,
    "concreteType": "TerraformCLIVersions",
    "kind": "LinkedField",
    "name": "terraformCLIVersions",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "versions",
        "storageKey": null
      }
    ],
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "TerraformCLIVersionSettingQuery",
    "selections": (v0/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "TerraformCLIVersionSettingQuery",
    "selections": (v0/*: any*/)
  },
  "params": {
    "cacheID": "be83d1f64aa7238c5cdd748c6ad9a8e0",
    "id": null,
    "metadata": {},
    "name": "TerraformCLIVersionSettingQuery",
    "operationKind": "query",
    "text": "query TerraformCLIVersionSettingQuery {\n  terraformCLIVersions {\n    versions\n  }\n}\n"
  }
};
})();

(node as any).hash = "f306af80a8b5db56eeb6f35f98f1c861";

export default node;
