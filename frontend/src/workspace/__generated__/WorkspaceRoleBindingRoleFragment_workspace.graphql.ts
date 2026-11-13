/**
 * @generated SignedSource<<1fab17eb02335c08ff2b1856a1a5de9a>>
 * @lightSyntaxTransform
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type WorkspaceRoleBindingRoleFragment_workspace$data = {
  readonly roleBinding: {
    readonly id: string;
    readonly role: {
      readonly description: string;
      readonly id: string;
      readonly name: string;
    };
  } | null | undefined;
  readonly " $fragmentType": "WorkspaceRoleBindingRoleFragment_workspace";
};
export type WorkspaceRoleBindingRoleFragment_workspace$key = {
  readonly " $data"?: WorkspaceRoleBindingRoleFragment_workspace$data;
  readonly " $fragmentSpreads": FragmentRefs<"WorkspaceRoleBindingRoleFragment_workspace">;
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
  "name": "WorkspaceRoleBindingRoleFragment_workspace",
  "selections": [
    {
      "alias": null,
      "args": null,
      "concreteType": "WorkspaceRoleBinding",
      "kind": "LinkedField",
      "name": "roleBinding",
      "plural": false,
      "selections": [
        (v0/*:: as any*/),
        {
          "alias": null,
          "args": null,
          "concreteType": "Role",
          "kind": "LinkedField",
          "name": "role",
          "plural": false,
          "selections": [
            (v0/*:: as any*/),
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
              "name": "description",
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": null
    }
  ],
  "type": "Workspace",
  "abstractKey": null
};
})();

(node as any).hash = "87e1077b28a914f890889758c1e9918c";

export default node;
