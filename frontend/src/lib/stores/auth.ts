import { writable, type Readable } from 'svelte/store';
import { browser } from '$app/environment';

export interface User {
  id: string;
  studentID: string;
  email: string;
  firstName: string;
  lastName: string;
  role: 'STUDENT' | 'SUPER_ADMIN' | 'FACULTY_ADMIN' | 'REGULAR_ADMIN';
  isActive: boolean;
  faculty?: {
    id: string;
    name: string;
    code: string;
  };
  department?: {
    id: string;
    name: string;
    code: string;
  };
}

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
}

const initialState: AuthState = {
  user: null,
  token: null,
  isAuthenticated: false,
  isLoading: true,
};

function createAuthStore() {
  const { subscribe, set, update } = writable<AuthState>(initialState);

  return {
    subscribe,
    
    // Initialize auth state from localStorage
    init: () => {
      if (browser) {
        const token = localStorage.getItem('token');
        const userStr = localStorage.getItem('user');
        
        if (token && userStr) {
          try {
            const user = JSON.parse(userStr);
            set({
              user,
              token,
              isAuthenticated: true,
              isLoading: false,
            });
          } catch (error) {
            console.error('Failed to parse user from localStorage:', error);
            localStorage.removeItem('token');
            localStorage.removeItem('user');
            set({ ...initialState, isLoading: false });
          }
        } else {
          set({ ...initialState, isLoading: false });
        }
      }
    },

    // Login
    login: (token: string, user: User) => {
      if (browser) {
        localStorage.setItem('token', token);
        localStorage.setItem('user', JSON.stringify(user));
      }
      
      set({
        user,
        token,
        isAuthenticated: true,
        isLoading: false,
      });
    },

    // Logout
    logout: () => {
      if (browser) {
        localStorage.removeItem('token');
        localStorage.removeItem('user');
      }
      
      set({
        user: null,
        token: null,
        isAuthenticated: false,
        isLoading: false,
      });
    },

    // Update user info
    updateUser: (user: User) => {
      if (browser) {
        localStorage.setItem('user', JSON.stringify(user));
      }
      
      update(state => ({
        ...state,
        user,
      }));
    },

    // Set loading state
    setLoading: (isLoading: boolean) => {
      update(state => ({
        ...state,
        isLoading,
      }));
    },
  };
}

export const authStore = createAuthStore();

// Derived stores for convenience
export const user: Readable<User | null> = {
  subscribe: (run) => authStore.subscribe(state => run(state.user))
};

export const isAuthenticated: Readable<boolean> = {
  subscribe: (run) => authStore.subscribe(state => run(state.isAuthenticated))
};

export const isAdmin: Readable<boolean> = {
  subscribe: (run) => authStore.subscribe(state => 
    run(state.user?.role !== 'STUDENT' || false)
  )
};

export const isSuperAdmin: Readable<boolean> = {
  subscribe: (run) => authStore.subscribe(state => 
    run(state.user?.role === 'SUPER_ADMIN' || false)
  )
};

export const isFacultyAdmin: Readable<boolean> = {
  subscribe: (run) => authStore.subscribe(state => 
    run(state.user?.role === 'FACULTY_ADMIN' || false)
  )
};