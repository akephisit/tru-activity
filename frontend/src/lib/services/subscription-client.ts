import { createClient, Client } from 'graphql-ws';
import { writable, type Writable } from 'svelte/stores';
import { browser } from '$app/environment';

export interface SubscriptionPayload {
  type: string;
  timestamp: string;
  data: any;
  metadata?: {
    source?: string;
    userId?: string;
    facultyId?: string;
    activityId?: string;
    correlationId?: string;
  };
}

export interface ConnectionStatus {
  connected: boolean;
  connecting: boolean;
  error: string | null;
  lastConnected: Date | null;
  reconnectAttempts: number;
  maxReconnectAttempts: number;
}

export interface SubscriptionFilter {
  facultyID?: string;
  activityID?: string;
  userID?: string;
  types?: string[];
}

export class SubscriptionClient {
  private client: Client | null = null;
  private url: string;
  private token: string | null = null;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private heartbeatTimer: NodeJS.Timeout | null = null;
  private subscriptions = new Map<string, () => void>();
  
  // Stores
  public connectionStatus: Writable<ConnectionStatus> = writable({
    connected: false,
    connecting: false,
    error: null,
    lastConnected: null,
    reconnectAttempts: 0,
    maxReconnectAttempts: 10
  });

  public personalNotifications: Writable<SubscriptionPayload[]> = writable([]);
  public systemAlerts: Writable<SubscriptionPayload[]> = writable([]);
  public activityUpdates: Writable<Map<string, SubscriptionPayload[]>> = writable(new Map());
  public qrScanEvents: Writable<SubscriptionPayload[]> = writable([]);
  public participationEvents: Writable<SubscriptionPayload[]> = writable([]);

  constructor(url: string) {
    this.url = url;
    
    if (browser) {
      // Auto-reconnect on page visibility change
      document.addEventListener('visibilitychange', () => {
        if (!document.hidden && !this.isConnected()) {
          this.connect(this.token);
        }
      });

      // Handle online/offline events
      window.addEventListener('online', () => {
        if (!this.isConnected()) {
          this.connect(this.token);
        }
      });

      window.addEventListener('offline', () => {
        this.updateConnectionStatus(status => ({
          ...status,
          connected: false,
          error: 'Network offline'
        }));
      });
    }
  }

  async connect(token: string | null = null): Promise<void> {
    if (!browser) return;

    this.token = token;
    
    this.updateConnectionStatus(status => ({
      ...status,
      connecting: true,
      error: null
    }));

    // Clear existing client
    if (this.client) {
      await this.disconnect();
    }

    try {
      this.client = createClient({
        url: this.url,
        connectionParams: () => ({
          authorization: token ? `Bearer ${token}` : undefined,
        }),
        shouldRetry: (errOrCloseEvent) => {
          // Retry on network errors and server errors
          if (errOrCloseEvent instanceof CloseEvent) {
            return errOrCloseEvent.code !== 1000 && errOrCloseEvent.code !== 1001;
          }
          return true;
        },
        retryAttempts: 5,
        retryWait: async (attempt) => {
          // Exponential backoff with jitter
          const delay = Math.min(1000 * Math.pow(2, attempt), 30000);
          const jitter = Math.random() * 0.3 * delay;
          await new Promise(resolve => setTimeout(resolve, delay + jitter));
        },
        onConnected: () => {
          console.log('WebSocket connected');
          this.updateConnectionStatus(status => ({
            ...status,
            connected: true,
            connecting: false,
            error: null,
            lastConnected: new Date(),
            reconnectAttempts: 0
          }));
          
          this.startHeartbeat();
          this.resubscribeAll();
        },
        onClosed: (event) => {
          console.log('WebSocket closed:', event);
          this.updateConnectionStatus(status => ({
            ...status,
            connected: false,
            connecting: false,
            error: event.reason || 'Connection closed'
          }));
          
          this.stopHeartbeat();
          this.scheduleReconnect();
        },
        onError: (error) => {
          console.error('WebSocket error:', error);
          this.updateConnectionStatus(status => ({
            ...status,
            error: error.message || 'Connection error'
          }));
        }
      });

    } catch (error) {
      console.error('Failed to create WebSocket client:', error);
      this.updateConnectionStatus(status => ({
        ...status,
        connecting: false,
        error: error instanceof Error ? error.message : 'Connection failed'
      }));
      
      this.scheduleReconnect();
    }
  }

  async disconnect(): Promise<void> {
    if (!browser) return;

    // Clear timers
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    
    this.stopHeartbeat();

    // Clear subscriptions
    this.subscriptions.clear();

    // Close client
    if (this.client) {
      await this.client.dispose();
      this.client = null;
    }

    this.updateConnectionStatus(status => ({
      ...status,
      connected: false,
      connecting: false,
      error: null
    }));
  }

  isConnected(): boolean {
    let status: ConnectionStatus;
    this.connectionStatus.subscribe(s => status = s)();
    return status!.connected;
  }

  // Subscription methods

  subscribeToPersonalNotifications(filter?: SubscriptionFilter): () => void {
    const subscriptionId = 'personal_notifications';
    
    const unsubscribe = this.createSubscription(
      subscriptionId,
      `
        subscription PersonalNotifications($filter: SubscriptionFilter) {
          personalNotifications(filter: $filter) {
            type
            timestamp
            data
            metadata {
              source
              userId
              facultyId
              activityId
              correlationId
            }
          }
        }
      `,
      { filter },
      (data) => {
        this.personalNotifications.update(notifications => [
          ...notifications.slice(-49), // Keep last 50 notifications
          data.personalNotifications
        ]);
      }
    );

    return unsubscribe;
  }

  subscribeToActivityUpdates(activityID: string): () => void {
    const subscriptionId = `activity_updates_${activityID}`;
    
    const unsubscribe = this.createSubscription(
      subscriptionId,
      `
        subscription ActivityUpdates($activityID: ID!) {
          activityUpdates(activityID: $activityID) {
            type
            timestamp
            data
            metadata {
              source
              activityId
            }
          }
        }
      `,
      { activityID },
      (data) => {
        this.activityUpdates.update(updates => {
          const newUpdates = new Map(updates);
          const activityUpdates = newUpdates.get(activityID) || [];
          newUpdates.set(activityID, [
            ...activityUpdates.slice(-19), // Keep last 20 updates per activity
            data.activityUpdates
          ]);
          return newUpdates;
        });
      }
    );

    return unsubscribe;
  }

  subscribeToSystemAlerts(filter?: SubscriptionFilter): () => void {
    const subscriptionId = 'system_alerts';
    
    const unsubscribe = this.createSubscription(
      subscriptionId,
      `
        subscription SystemAlerts($filter: SubscriptionFilter) {
          systemAlerts(filter: $filter) {
            type
            timestamp
            data
            metadata {
              source
            }
          }
        }
      `,
      { filter },
      (data) => {
        this.systemAlerts.update(alerts => [
          ...alerts.slice(-29), // Keep last 30 alerts
          data.systemAlerts
        ]);
      }
    );

    return unsubscribe;
  }

  subscribeToQRScanEvents(activityID?: string): () => void {
    const subscriptionId = `qr_scan_events_${activityID || 'all'}`;
    
    const unsubscribe = this.createSubscription(
      subscriptionId,
      `
        subscription QRScanEvents($activityID: ID) {
          qrScanEvents(activityID: $activityID) {
            type
            timestamp
            data
            metadata {
              source
              activityId
              userId
            }
          }
        }
      `,
      { activityID },
      (data) => {
        this.qrScanEvents.update(events => [
          ...events.slice(-49), // Keep last 50 scan events
          data.qrScanEvents
        ]);
      }
    );

    return unsubscribe;
  }

  subscribeToParticipationEvents(activityID?: string, userID?: string): () => void {
    const subscriptionId = `participation_events_${activityID || 'all'}_${userID || 'all'}`;
    
    const unsubscribe = this.createSubscription(
      subscriptionId,
      `
        subscription ParticipationEvents($activityID: ID, $userID: ID) {
          participationEvents(activityID: $activityID, userID: $userID) {
            type
            timestamp
            data
            metadata {
              source
              activityId
              userId
            }
          }
        }
      `,
      { activityID, userID },
      (data) => {
        this.participationEvents.update(events => [
          ...events.slice(-49), // Keep last 50 participation events
          data.participationEvents
        ]);
      }
    );

    return unsubscribe;
  }

  subscribeToFacultyUpdates(facultyID: string): () => void {
    const subscriptionId = `faculty_updates_${facultyID}`;
    
    const unsubscribe = this.createSubscription(
      subscriptionId,
      `
        subscription FacultyUpdates($facultyID: ID!) {
          facultyUpdates(facultyID: $facultyID) {
            type
            timestamp
            data
            metadata {
              source
              facultyId
            }
          }
        }
      `,
      { facultyID },
      (data) => {
        // Handle faculty updates (could add to a separate store if needed)
        console.log('Faculty update received:', data.facultyUpdates);
      }
    );

    return unsubscribe;
  }

  subscribeToHeartbeat(): () => void {
    const subscriptionId = 'heartbeat';
    
    const unsubscribe = this.createSubscription(
      subscriptionId,
      `
        subscription Heartbeat {
          heartbeat
        }
      `,
      {},
      (data) => {
        console.log('Heartbeat received:', data.heartbeat);
        // Update last activity timestamp
        this.updateConnectionStatus(status => ({
          ...status,
          lastConnected: new Date()
        }));
      }
    );

    return unsubscribe;
  }

  // Private methods

  private createSubscription(
    id: string,
    query: string,
    variables: any,
    onData: (data: any) => void
  ): () => void {
    if (!this.client) {
      throw new Error('Client not connected');
    }

    const unsubscribe = this.client.subscribe(
      {
        query,
        variables
      },
      {
        next: (data) => {
          onData(data.data);
        },
        error: (error) => {
          console.error(`Subscription ${id} error:`, error);
          this.updateConnectionStatus(status => ({
            ...status,
            error: `Subscription error: ${error.message}`
          }));
        },
        complete: () => {
          console.log(`Subscription ${id} completed`);
          this.subscriptions.delete(id);
        }
      }
    );

    // Store subscription for resubscription on reconnect
    this.subscriptions.set(id, () => {
      this.createSubscription(id, query, variables, onData);
    });

    return () => {
      unsubscribe();
      this.subscriptions.delete(id);
    };
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) return;

    this.connectionStatus.subscribe(status => {
      if (status.reconnectAttempts >= status.maxReconnectAttempts) {
        console.log('Max reconnection attempts reached');
        return;
      }

      const delay = Math.min(1000 * Math.pow(2, status.reconnectAttempts), 30000);
      
      this.reconnectTimer = setTimeout(() => {
        this.updateConnectionStatus(s => ({
          ...s,
          reconnectAttempts: s.reconnectAttempts + 1
        }));
        
        this.connect(this.token);
        this.reconnectTimer = null;
      }, delay);
    })();
  }

  private resubscribeAll(): void {
    // Resubscribe to all active subscriptions
    const subscriptionsToRestore = Array.from(this.subscriptions.values());
    this.subscriptions.clear();
    
    subscriptionsToRestore.forEach(resubscribe => {
      try {
        resubscribe();
      } catch (error) {
        console.error('Failed to resubscribe:', error);
      }
    });
  }

  private startHeartbeat(): void {
    this.stopHeartbeat();
    
    // Send heartbeat every 30 seconds
    this.heartbeatTimer = setInterval(() => {
      // The heartbeat subscription handles this automatically
    }, 30000);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private updateConnectionStatus(updater: (status: ConnectionStatus) => ConnectionStatus): void {
    this.connectionStatus.update(updater);
  }

  // Utility methods

  clearNotifications(): void {
    this.personalNotifications.set([]);
    this.systemAlerts.set([]);
    this.activityUpdates.set(new Map());
    this.qrScanEvents.set([]);
    this.participationEvents.set([]);
  }

  getConnectionStats(): Promise<ConnectionStatus> {
    return new Promise(resolve => {
      this.connectionStatus.subscribe(status => resolve(status))();
    });
  }
}

// Global singleton instance
let subscriptionClient: SubscriptionClient | null = null;

export function getSubscriptionClient(): SubscriptionClient {
  if (!subscriptionClient && browser) {
    const wsUrl = import.meta.env.VITE_GRAPHQL_WS_URL || 'ws://localhost:8080/graphql';
    subscriptionClient = new SubscriptionClient(wsUrl);
  }
  return subscriptionClient!;
}

// Helper function to format subscription payloads for display
export function formatSubscriptionPayload(payload: SubscriptionPayload): string {
  const timestamp = new Date(payload.timestamp).toLocaleTimeString();
  return `[${timestamp}] ${payload.type}: ${JSON.stringify(payload.data)}`;
}

// Helper function to get notification icon based on type
export function getNotificationIcon(type: string): string {
  const iconMap: Record<string, string> = {
    'qr_scan_event': '📱',
    'participation_event': '✅',
    'activity_update': '📅',
    'system_alert': '🚨',
    'personal_notification': '🔔',
    'faculty_update': '🏫',
    'subscription_warning': '⚠️'
  };
  
  return iconMap[type] || '📬';
}