import { createClient, ssrExchange, cacheExchange, fetchExchange } from '@urql/svelte';
import { browser } from '$app/environment';
import { PUBLIC_API_URL, PUBLIC_GRAPHQL_URL } from '$env/static/public';

// Auth token management
let authToken: string | null = null;

export function setAuthToken(token: string | null) {
  authToken = token;
  if (browser) {
    if (token) {
      localStorage.setItem('token', token);
    } else {
      localStorage.removeItem('token');
    }
  }
}

export function getAuthToken(): string | null {
  if (authToken) return authToken;
  
  if (browser) {
    authToken = localStorage.getItem('token');
    return authToken;
  }
  
  return null;
}

// Get GraphQL URL from environment variables with fallback
const getGraphQLURL = () => {
  if (browser) {
    return PUBLIC_GRAPHQL_URL || 'http://localhost:8080/query';
  }
  return 'http://backend:8080/query'; // Server-side fallback
};

// SSR exchange for server-side rendering
const ssr = ssrExchange({
  isClient: browser,
});

// Create URQL client
export const client = createClient({
  url: getGraphQLURL(),
  exchanges: [
    cacheExchange,
    ssr,
    fetchExchange,
  ],
  fetchOptions: () => {
    const token = getAuthToken();
    return {
      headers: {
        authorization: token ? `Bearer ${token}` : '',
      },
    };
  },
});

// Export SSR exchange for use in app.html
export { ssr };

// Utility functions for common operations
export async function graphqlRequest<T = any>(
  query: string,
  variables?: any,
  options?: RequestInit
): Promise<T> {
  const token = getAuthToken();
  
  const response = await fetch(getGraphQLURL(), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': token ? `Bearer ${token}` : '',
      ...options?.headers,
    },
    body: JSON.stringify({
      query,
      variables,
    }),
    ...options,
  });

  const result = await response.json();

  if (result.errors) {
    // Handle authentication errors
    const authError = result.errors.find((error: any) => 
      error.message?.includes('authentication') || 
      error.message?.includes('unauthorized')
    );
    
    if (authError && browser) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    
    throw new Error(result.errors[0]?.message || 'GraphQL Error');
  }

  return result.data;
}

// SSE Client for real-time events
export class SSEClient {
  private eventSource: EventSource | null = null;
  private token: string | null = null;

  constructor(token?: string) {
    this.token = token || getAuthToken();
  }

  connect(): EventSource {
    if (!browser) {
      throw new Error('SSE is only available in the browser');
    }

    if (this.eventSource) {
      this.disconnect();
    }

    const url = new URL('/sse/events', PUBLIC_API_URL);
    if (this.token) {
      url.searchParams.set('token', this.token);
    }

    this.eventSource = new EventSource(url.toString());
    return this.eventSource;
  }

  disconnect() {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
  }

  subscribe(eventType: string, filter?: any): Promise<Response> {
    if (!this.token) {
      throw new Error('Authentication token required');
    }

    return fetch(`${PUBLIC_API_URL}/sse/subscribe`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.token}`,
        'X-Client-ID': 'svelte-client-' + Date.now(),
      },
      body: JSON.stringify({
        eventType,
        filter,
      }),
    });
  }

  unsubscribe(eventType: string): Promise<Response> {
    if (!this.token) {
      throw new Error('Authentication token required');
    }

    return fetch(`${PUBLIC_API_URL}/sse/unsubscribe`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.token}`,
        'X-Client-ID': 'svelte-client-' + Date.now(),
      },
      body: JSON.stringify({
        eventType,
      }),
    });
  }
}

// Create global SSE client instance
export const sseClient = browser ? new SSEClient() : null;