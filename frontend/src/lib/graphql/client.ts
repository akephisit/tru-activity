import { ApolloClient, InMemoryCache, createHttpLink, from } from '@apollo/client';
import { setContext } from '@apollo/client/link/context';
import { onError } from '@apollo/client/link/error';
import { browser } from '$app/environment';
import { env } from '$env/dynamic/public';

// Get GraphQL URL from environment variables with fallback
const getGraphQLURL = () => {
  if (browser) {
    return env.PUBLIC_GRAPHQL_URL || 'http://localhost:8080/query';
  }
  return 'http://backend:8080/query'; // Server-side fallback
};

// HTTP Link
const httpLink = createHttpLink({
  uri: getGraphQLURL(),
});

// Auth Link
const authLink = setContext((_, { headers }) => {
  const token = browser ? localStorage.getItem('token') : null;

  return {
    headers: {
      ...headers,
      authorization: token ? `Bearer ${token}` : "",
    }
  };
});

// Error Link
const errorLink = onError(({ graphQLErrors, networkError, operation, forward }) => {
  if (graphQLErrors) {
    graphQLErrors.forEach(({ message, locations, path }) => {
      console.error(
        `[GraphQL error]: Message: ${message}, Location: ${locations}, Path: ${path}`,
      );
    });
  }

  if (networkError) {
    console.error(`[Network error]: ${networkError}`);
    
    // Handle authentication errors
    if ('statusCode' in networkError && networkError.statusCode === 401) {
      if (browser) {
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        window.location.href = '/login';
      }
    }
  }
});

// Apollo Client
export const client = new ApolloClient({
  link: from([errorLink, authLink, httpLink]),
  cache: new InMemoryCache({
    typePolicies: {
      User: {
        fields: {
          participations: {
            merge(existing = [], incoming) {
              return incoming;
            },
          },
          subscriptions: {
            merge(existing = [], incoming) {
              return incoming;
            },
          },
        },
      },
      Activity: {
        fields: {
          participations: {
            merge(existing = [], incoming) {
              return incoming;
            },
          },
        },
      },
    },
  }),
  defaultOptions: {
    watchQuery: {
      errorPolicy: 'all',
    },
    query: {
      errorPolicy: 'all',
    },
  },
});