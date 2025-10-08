import AuthenticationService from "../auth/AuthenticationService";
import cfg from '../common/config';

class GraphQLError extends Error {
    codes: string[]
    constructor(message: string, codes: string[]) {
        super(message);
        this.codes = codes;
    }
  }

const graphQLFetcher = (authService: AuthenticationService) => {
  return async (text: string, variables?: object) => {
    const response = await authService.fetchWithAuth(`${cfg.apiUrl}/graphql`, { // nosemgrep: nodejs_scan.javascript-ssrf-rule-node_ssrf
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        query: text,
        variables,
      }),
    });

    // Get the response as JSON
    const json = await response.json();
    // Throw error here so it'll be caught by error boundary
    if (json.errors && json.errors.length) {
        throw new GraphQLError(`GraphQL query failed: ${json.errors.map((e: any) => e.message).join('; ')}`, json.errors.map((e: any) => e.extensions.code))
    }
    return json;
  }
}

export default graphQLFetcher;
