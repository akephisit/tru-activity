import { writable, type Writable } from 'svelte/store';
import { browser } from '$app/environment';
import { env } from '$env/dynamic/public';

export interface SSEEvent {
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

export interface SSEFilter {
  facultyID?: string;
  activityID?: string;
  userID?: string;
  types?: string[];
}

export class SSEClient {
  private eventSource: EventSource | null = null;
  private baseUrl: string;
  private token: string | null = null;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private heartbeatTimer: NodeJS.Timeout | null = null;
  private subscriptions = new Set<string>();
  
  // Stores
  public connectionStatus: Writable<ConnectionStatus> = writable({
    connected: false,
    connecting: false,
    error: null,
    lastConnected: null,
    reconnectAttempts: 0,
    maxReconnectAttempts: 10
  });

  public personalNotifications: Writable<SSEEvent[]> = writable([]);
  public systemAlerts: Writable<SSEEvent[]> = writable([]);
  public activityUpdates: Writable<Map<string, SSEEvent[]>> = writable(new Map());
  public qrScanEvents: Writable<SSEEvent[]> = writable([]);
  public participationEvents: Writable<SSEEvent[]> = writable([]);
  public facultyUpdates: Writable<SSEEvent[]> = writable([]);

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
    
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

    // Close existing connection
    if (this.eventSource) {
      this.disconnect();
    }

    try {
      const url = new URL('/events', this.baseUrl);
      if (token) {
        url.searchParams.set('token', token);
      }

      this.eventSource = new EventSource(url.toString());

      this.eventSource.onopen = () => {
        console.log('SSE connection opened');
        this.updateConnectionStatus(status => ({
          ...status,
          connected: true,
          connecting: false,
          error: null,
          lastConnected: new Date(),
          reconnectAttempts: 0
        }));
        
        this.startHeartbeat();
      };

      this.eventSource.onmessage = (event) => {
        try {
          const sseEvent: SSEEvent = JSON.parse(event.data);
          this.handleEvent(sseEvent);
        } catch (error) {
          console.error('Failed to parse SSE event:', error);
        }
      };

      this.eventSource.onerror = (error) => {
        console.error('SSE connection error:', error);
        this.updateConnectionStatus(status => ({
          ...status,
          connected: false,
          connecting: false,
          error: 'Connection error'
        }));
        
        this.stopHeartbeat();
        this.scheduleReconnect();
      };

      // Setup event listeners for specific event types
      this.setupEventListeners();

    } catch (error) {
      console.error('Failed to create SSE connection:', error);
      this.updateConnectionStatus(status => ({
        ...status,
        connecting: false,
        error: error instanceof Error ? error.message : 'Connection failed'
      }));
      
      this.scheduleReconnect();
    }
  }

  private setupEventListeners(): void {
    if (!this.eventSource) return;

    // Personal notifications
    this.eventSource.addEventListener('personal_notification', (event) => {
      const sseEvent: SSEEvent = JSON.parse(event.data);
      this.personalNotifications.update(notifications => [
        ...notifications.slice(-49), // Keep last 50 notifications
        sseEvent
      ]);
    });

    // System alerts
    this.eventSource.addEventListener('system_alert', (event) => {
      const sseEvent: SSEEvent = JSON.parse(event.data);
      this.systemAlerts.update(alerts => [
        ...alerts.slice(-29), // Keep last 30 alerts
        sseEvent
      ]);
    });

    // Activity updates
    this.eventSource.addEventListener('activity_update', (event) => {
      const sseEvent: SSEEvent = JSON.parse(event.data);
      const activityId = sseEvent.metadata?.activityId;
      if (activityId) {
        this.activityUpdates.update(updates => {
          const newUpdates = new Map(updates);
          const activityUpdates = newUpdates.get(activityId) || [];
          newUpdates.set(activityId, [
            ...activityUpdates.slice(-19), // Keep last 20 updates per activity
            sseEvent
          ]);
          return newUpdates;
        });
      }
    });

    // QR scan events
    this.eventSource.addEventListener('qr_scan_event', (event) => {
      const sseEvent: SSEEvent = JSON.parse(event.data);
      this.qrScanEvents.update(events => [
        ...events.slice(-49), // Keep last 50 scan events
        sseEvent
      ]);
    });

    // Participation events
    this.eventSource.addEventListener('participation_event', (event) => {
      const sseEvent: SSEEvent = JSON.parse(event.data);
      this.participationEvents.update(events => [
        ...events.slice(-49), // Keep last 50 participation events
        sseEvent
      ]);
    });

    // Faculty updates
    this.eventSource.addEventListener('faculty_update', (event) => {
      const sseEvent: SSEEvent = JSON.parse(event.data);
      this.facultyUpdates.update(updates => [
        ...updates.slice(-29), // Keep last 30 faculty updates
        sseEvent
      ]);
    });

    // Heartbeat
    this.eventSource.addEventListener('heartbeat', (event) => {
      console.log('Heartbeat received:', event.data);
      this.updateConnectionStatus(status => ({
        ...status,
        lastConnected: new Date()
      }));
    });
  }

  private handleEvent(sseEvent: SSEEvent): void {
    // Handle generic events that don't have specific event types
    console.log('Received SSE event:', sseEvent);
    
    // Update last activity timestamp
    this.updateConnectionStatus(status => ({
      ...status,
      lastConnected: new Date()
    }));
  }

  disconnect(): void {
    if (!browser) return;

    // Clear timers
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    
    this.stopHeartbeat();

    // Clear subscriptions
    this.subscriptions.clear();

    // Close connection
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
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

  // Subscription management methods
  subscribeToPersonalNotifications(filter?: SSEFilter): () => void {
    const subscriptionId = 'personal_notifications';
    this.subscriptions.add(subscriptionId);
    
    // Send subscription request to backend
    this.sendSubscriptionRequest('personal_notifications', filter);

    return () => {
      this.subscriptions.delete(subscriptionId);
      this.sendUnsubscriptionRequest('personal_notifications');
    };
  }

  subscribeToActivityUpdates(activityId: string): () => void {
    const subscriptionId = `activity_updates_${activityId}`;
    this.subscriptions.add(subscriptionId);
    
    this.sendSubscriptionRequest('activity_updates', { activityID: activityId });

    return () => {
      this.subscriptions.delete(subscriptionId);
      this.sendUnsubscriptionRequest('activity_updates', { activityID: activityId });
    };
  }

  subscribeToSystemAlerts(filter?: SSEFilter): () => void {
    const subscriptionId = 'system_alerts';
    this.subscriptions.add(subscriptionId);
    
    this.sendSubscriptionRequest('system_alerts', filter);

    return () => {
      this.subscriptions.delete(subscriptionId);
      this.sendUnsubscriptionRequest('system_alerts');
    };
  }

  subscribeToQRScanEvents(activityId?: string): () => void {
    const subscriptionId = `qr_scan_events_${activityId || 'all'}`;
    this.subscriptions.add(subscriptionId);
    
    this.sendSubscriptionRequest('qr_scan_events', activityId ? { activityID: activityId } : undefined);

    return () => {
      this.subscriptions.delete(subscriptionId);
      this.sendUnsubscriptionRequest('qr_scan_events', activityId ? { activityID: activityId } : undefined);
    };
  }

  subscribeToParticipationEvents(activityId?: string, userId?: string): () => void {
    const subscriptionId = `participation_events_${activityId || 'all'}_${userId || 'all'}`;
    this.subscriptions.add(subscriptionId);
    
    this.sendSubscriptionRequest('participation_events', { 
      activityID: activityId, 
      userID: userId 
    });

    return () => {
      this.subscriptions.delete(subscriptionId);
      this.sendUnsubscriptionRequest('participation_events', { 
        activityID: activityId, 
        userID: userId 
      });
    };
  }

  subscribeToFacultyUpdates(facultyId: string): () => void {
    const subscriptionId = `faculty_updates_${facultyId}`;
    this.subscriptions.add(subscriptionId);
    
    this.sendSubscriptionRequest('faculty_updates', { facultyID: facultyId });

    return () => {
      this.subscriptions.delete(subscriptionId);
      this.sendUnsubscriptionRequest('faculty_updates', { facultyID: facultyId });
    };
  }

  // Private methods
  private sendSubscriptionRequest(eventType: string, filter?: any): void {
    // Since SSE is unidirectional, we can use a separate HTTP request to register subscriptions
    // Or include subscription info in the initial connection URL
    if (this.token) {
      fetch(`${this.baseUrl}/events/subscribe`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ eventType, filter })
      }).catch(error => {
        console.error('Failed to send subscription request:', error);
      });
    }
  }

  private sendUnsubscriptionRequest(eventType: string, filter?: any): void {
    if (this.token) {
      fetch(`${this.baseUrl}/events/unsubscribe`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ eventType, filter })
      }).catch(error => {
        console.error('Failed to send unsubscription request:', error);
      });
    }
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

  private startHeartbeat(): void {
    this.stopHeartbeat();
    
    // Send heartbeat request every 30 seconds
    this.heartbeatTimer = setInterval(() => {
      if (this.token) {
        fetch(`${this.baseUrl}/events/heartbeat`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${this.token}`
          }
        }).catch(error => {
          console.error('Heartbeat failed:', error);
        });
      }
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
    this.facultyUpdates.set([]);
  }

  getConnectionStats(): Promise<ConnectionStatus> {
    return new Promise(resolve => {
      this.connectionStatus.subscribe(status => resolve(status))();
    });
  }
}

// Global singleton instance
let sseClient: SSEClient | null = null;

export function getSSEClient(): SSEClient {
  if (!sseClient && browser) {
    const baseUrl = env.PUBLIC_API_URL || 'http://localhost:8080';
    sseClient = new SSEClient(baseUrl);
  }
  return sseClient!;
}

// Helper function to format SSE events for display
export function formatSSEEvent(event: SSEEvent): string {
  const timestamp = new Date(event.timestamp).toLocaleTimeString();
  return `[${timestamp}] ${event.type}: ${JSON.stringify(event.data)}`;
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