/**
 * @generated SignedSource<<a7bc77a077c77f7b35a908184353cb0a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProblemType = "BAD_REQUEST" | "CONFLICT" | "FORBIDDEN" | "NOT_FOUND" | "SERVICE_UNAVAILABLE" | "%future added value";
export type UpdateWorkspaceVCSProviderLinkInput = {
  autoSpeculativePlan?: boolean | null | undefined;
  branch?: string | null | undefined;
  clientMutationId?: string | null | undefined;
  globPatterns: ReadonlyArray<string>;
  id: string;
  metadata?: ResourceMetadataInput | null | undefined;
  moduleDirectory?: string | null | undefined;
  tagRegex?: string | null | undefined;
  webhookDisabled?: boolean | null | undefined;
};
export type ResourceMetadataInput = {
  version: string;
};
export type EditVCSProviderLinkMutation$variables = {
  input: UpdateWorkspaceVCSProviderLinkInput;
};
export type EditVCSProviderLinkMutation$data = {
  readonly updateWorkspaceVCSProviderLink: {
    readonly problems: ReadonlyArray<{
      readonly field: ReadonlyArray<string> | null | undefined;
      readonly message: string;
      readonly type: ProblemType;
    }>;
    readonly vcsProviderLink: {
      readonly autoSpeculativePlan: boolean;
      readonly branch: string;
      readonly globPatterns: ReadonlyArray<string>;
      readonly id: string;
      readonly moduleDirectory: string | null | undefined;
      readonly repositoryPath: string;
      readonly tagRegex: string | null | undefined;
      readonly webhookDisabled: boolean;
    } | null | undefined;
  };
};
export type EditVCSProviderLinkMutation = {
  response: EditVCSProviderLinkMutation$data;
  variables: EditVCSProviderLinkMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "input"
  }
],
v1 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "input",
        "variableName": "input"
      }
    ],
    "concreteType": "UpdateWorkspaceVCSProviderLinkPayload",
    "kind": "LinkedField",
    "name": "updateWorkspaceVCSProviderLink",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "WorkspaceVCSProviderLink",
        "kind": "LinkedField",
        "name": "vcsProviderLink",
        "plural": false,
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
            "name": "repositoryPath",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "moduleDirectory",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "branch",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "tagRegex",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "globPatterns",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "autoSpeculativePlan",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "webhookDisabled",
            "storageKey": null
          }
        ],
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "concreteType": "Problem",
        "kind": "LinkedField",
        "name": "problems",
        "plural": true,
        "selections": [
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "message",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "field",
            "storageKey": null
          },
          {
            "alias": null,
            "args": null,
            "kind": "ScalarField",
            "name": "type",
            "storageKey": null
          }
        ],
        "storageKey": null
      }
    ],
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "EditVCSProviderLinkMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "EditVCSProviderLinkMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "b07cfff21ecac1f031c35b59a951b304",
    "id": null,
    "metadata": {},
    "name": "EditVCSProviderLinkMutation",
    "operationKind": "mutation",
    "text": "mutation EditVCSProviderLinkMutation(\n  $input: UpdateWorkspaceVCSProviderLinkInput!\n) {\n  updateWorkspaceVCSProviderLink(input: $input) {\n    vcsProviderLink {\n      id\n      repositoryPath\n      moduleDirectory\n      branch\n      tagRegex\n      globPatterns\n      autoSpeculativePlan\n      webhookDisabled\n    }\n    problems {\n      message\n      field\n      type\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "9c6a75e9e7f8afe9136d989cc8d9d7d4";

export default node;
