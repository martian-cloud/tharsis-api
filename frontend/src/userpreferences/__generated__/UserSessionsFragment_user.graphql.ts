/**
 * @generated SignedSource<<f1af93d0581ae2de68984d6c2226e945>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ReaderFragment } from 'relay-runtime';
import { FragmentRefs } from "relay-runtime";
export type UserSessionsFragment_user$data = {
  readonly id: string;
  readonly userSessions: {
    readonly edges: ReadonlyArray<{
      readonly node: {
        readonly id: string;
        readonly " $fragmentSpreads": FragmentRefs<"UserSessionFragment_session">;
      } | null | undefined;
    } | null | undefined> | null | undefined;
  };
  readonly " $fragmentType": "UserSessionsFragment_user";
};
export type UserSessionsFragment_user$key = {
  readonly " $data"?: UserSessionsFragment_user$data;
  readonly " $fragmentSpreads": FragmentRefs<"UserSessionsFragment_user">;
};

import UserSessionsPaginationQuery_graphql from './UserSessionsPaginationQuery.graphql';

const node: ReaderFragment = (function(){
var v0 = [
  "userSessions"
],
v1 = {
  "alias": null,
  "args": null,
  "kind": "ScalarField",
  "name": "id",
  "storageKey": null
};
return {
  "argumentDefinitions": [
    {
      "kind": "RootArgument",
      "name": "after"
    },
    {
      "kind": "RootArgument",
      "name": "first"
    }
  ],
  "kind": "Fragment",
  "metadata": {
    "connection": [
      {
        "count": "first",
        "cursor": "after",
        "direction": "forward",
        "path": (v0/*: any*/)
      }
    ],
    "refetch": {
      "connection": {
        "forward": {
          "count": "first",
          "cursor": "after"
        },
        "backward": null,
        "path": (v0/*: any*/)
      },
      "fragmentPathInResult": [
        "node"
      ],
      "operation": UserSessionsPaginationQuery_graphql,
      "identifierInfo": {
        "identifierField": "id",
        "identifierQueryVariableName": "id"
      }
    }
  },
  "name": "UserSessionsFragment_user",
  "selections": [
    {
      "alias": "userSessions",
      "args": [
        {
          "kind": "Literal",
          "name": "sort",
          "value": "CREATED_AT_DESC"
        }
      ],
      "concreteType": "UserSessionConnection",
      "kind": "LinkedField",
      "name": "__UserSessions_userSessions_connection",
      "plural": false,
      "selections": [
        {
          "alias": null,
          "args": null,
          "concreteType": "UserSessionEdge",
          "kind": "LinkedField",
          "name": "edges",
          "plural": true,
          "selections": [
            {
              "alias": null,
              "args": null,
              "concreteType": "UserSession",
              "kind": "LinkedField",
              "name": "node",
              "plural": false,
              "selections": [
                (v1/*: any*/),
                {
                  "args": null,
                  "kind": "FragmentSpread",
                  "name": "UserSessionFragment_session"
                },
                {
                  "alias": null,
                  "args": null,
                  "kind": "ScalarField",
                  "name": "__typename",
                  "storageKey": null
                }
              ],
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "cursor",
              "storageKey": null
            }
          ],
          "storageKey": null
        },
        {
          "alias": null,
          "args": null,
          "concreteType": "PageInfo",
          "kind": "LinkedField",
          "name": "pageInfo",
          "plural": false,
          "selections": [
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "endCursor",
              "storageKey": null
            },
            {
              "alias": null,
              "args": null,
              "kind": "ScalarField",
              "name": "hasNextPage",
              "storageKey": null
            }
          ],
          "storageKey": null
        }
      ],
      "storageKey": "__UserSessions_userSessions_connection(sort:\"CREATED_AT_DESC\")"
    },
    (v1/*: any*/)
  ],
  "type": "User",
  "abstractKey": null
};
})();

(node as any).hash = "9ed49a905409e92e53a7351422c81666";

export default node;
