<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { fade, fly } from 'svelte/transition';
  import { 
    getSubscriptionClient, 
    formatSubscriptionPayload, 
    getNotificationIcon,
    type SubscriptionPayload 
  } from '$lib/services/subscription-client';
  import { Button } from '$lib/components/ui/button';
  import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { 
    Bell, 
    BellOff, 
    X, 
    Wifi, 
    WifiOff, 
    AlertTriangle,
    CheckCircle,
    Settings,
    Trash2
  } from 'lucide-svelte';

  export let position: 'top-right' | 'top-left' | 'bottom-right' | 'bottom-left' = 'top-right';
  export let maxNotifications = 10;
  export let autoHideDelay = 5000; // 5 seconds
  export let showConnectionStatus = true;
  export let enableSound = false;

  const client = getSubscriptionClient();
  
  let isOpen = false;
  let notifications: (SubscriptionPayload & { id: string; autoHide?: boolean })[] = [];
  let unsubscribeFunctions: (() => void)[] = [];
  let connectionStatus = { connected: false, connecting: false, error: null };
  let notificationCount = 0;

  // Audio for notifications
  let notificationSound: HTMLAudioElement;

  onMount(() => {
    // Initialize audio
    if (enableSound) {
      notificationSound = new Audio('/notification-sound.mp3');
      notificationSound.volume = 0.3;
    }

    // Subscribe to connection status
    const unsubscribeStatus = client.connectionStatus.subscribe(status => {
      connectionStatus = status;
    });

    // Subscribe to personal notifications
    const unsubscribePersonal = client.personalNotifications.subscribe(personalNotifications => {
      personalNotifications.forEach(payload => {
        addNotification(payload);
      });
    });

    // Subscribe to system alerts
    const unsubscribeAlerts = client.systemAlerts.subscribe(alerts => {
      alerts.forEach(payload => {
        addNotification(payload, false); // System alerts don't auto-hide
      });
    });

    // Subscribe to QR scan events (for admins)
    const unsubscribeQR = client.qrScanEvents.subscribe(events => {
      events.forEach(payload => {
        addNotification(payload);
      });
    });

    // Subscribe to participation events
    const unsubscribeParticipation = client.participationEvents.subscribe(events => {
      events.forEach(payload => {
        addNotification(payload);
      });
    });

    unsubscribeFunctions = [
      unsubscribeStatus,
      unsubscribePersonal,
      unsubscribeAlerts,
      unsubscribeQR,
      unsubscribeParticipation
    ];
  });

  onDestroy(() => {
    unsubscribeFunctions.forEach(fn => fn());
  });

  function addNotification(payload: SubscriptionPayload, autoHide = true) {
    const notification = {
      ...payload,
      id: `${Date.now()}-${Math.random()}`,
      autoHide
    };

    notifications = [notification, ...notifications.slice(0, maxNotifications - 1)];
    notificationCount++;

    // Play notification sound
    if (enableSound && notificationSound) {
      notificationSound.play().catch(console.warn);
    }

    // Auto-hide notification
    if (autoHide && autoHideDelay > 0) {
      setTimeout(() => {
        removeNotification(notification.id);
      }, autoHideDelay);
    }
  }

  function removeNotification(id: string) {
    notifications = notifications.filter(n => n.id !== id);
  }

  function clearAllNotifications() {
    notifications = [];
    notificationCount = 0;
  }

  function toggleNotificationCenter() {
    isOpen = !isOpen;
    if (isOpen) {
      notificationCount = 0; // Reset unread count when opened
    }
  }

  function getNotificationTypeColor(type: string): string {
    const colorMap: Record<string, string> = {
      'system_alert': 'destructive',
      'qr_scan_event': 'default',
      'participation_event': 'secondary',
      'activity_update': 'outline',
      'personal_notification': 'default',
      'faculty_update': 'secondary',
      'subscription_warning': 'destructive'
    };
    return colorMap[type] || 'outline';
  }

  function getNotificationPriority(type: string): 'high' | 'medium' | 'low' {
    const priorityMap: Record<string, 'high' | 'medium' | 'low'> = {
      'system_alert': 'high',
      'subscription_warning': 'high',
      'qr_scan_event': 'medium',
      'participation_event': 'medium',
      'activity_update': 'low',
      'personal_notification': 'low',
      'faculty_update': 'low'
    };
    return priorityMap[type] || 'low';
  }

  function formatTimestamp(timestamp: string): string {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    return date.toLocaleDateString();
  }

  $: positionClasses = {
    'top-right': 'top-4 right-4',
    'top-left': 'top-4 left-4',
    'bottom-right': 'bottom-4 right-4',
    'bottom-left': 'bottom-4 left-4'
  }[position];

  $: hasUnreadNotifications = notificationCount > 0;
  $: sortedNotifications = notifications.sort((a, b) => {
    const priorityOrder = { high: 3, medium: 2, low: 1 };
    const priorityA = priorityOrder[getNotificationPriority(a.type)];
    const priorityB = priorityOrder[getNotificationPriority(b.type)];
    
    if (priorityA !== priorityB) return priorityB - priorityA;
    return new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime();
  });
</script>

<!-- Notification Bell Button -->
<div class="fixed {positionClasses} z-50">
  <div class="relative">
    <!-- Connection Status Indicator -->
    {#if showConnectionStatus}
      <div class="absolute -top-1 -left-1 z-10">
        {#if connectionStatus.connected}
          <div class="w-3 h-3 bg-green-500 rounded-full animate-pulse" title="Connected"></div>
        {:else if connectionStatus.connecting}
          <div class="w-3 h-3 bg-yellow-500 rounded-full animate-pulse" title="Connecting..."></div>
        {:else}
          <div class="w-3 h-3 bg-red-500 rounded-full" title="Disconnected"></div>
        {/if}
      </div>
    {/if}

    <!-- Notification Bell -->
    <Button 
      variant="outline" 
      size="icon"
      on:click={toggleNotificationCenter}
      class="relative h-10 w-10 rounded-full shadow-lg {hasUnreadNotifications ? 'ring-2 ring-blue-500 ring-offset-2' : ''}"
    >
      {#if hasUnreadNotifications}
        <Bell size={20} class="text-blue-600" />
      {:else}
        <BellOff size={20} class="text-gray-500" />
      {/if}
      
      <!-- Notification Count Badge -->
      {#if notificationCount > 0}
        <span 
          class="absolute -top-2 -right-2 h-5 w-5 rounded-full bg-red-500 text-white text-xs flex items-center justify-center font-bold"
          transition:fly={{ y: -10, duration: 200 }}
        >
          {notificationCount > 99 ? '99+' : notificationCount}
        </span>
      {/if}
    </Button>
  </div>

  <!-- Notification Panel -->
  {#if isOpen}
    <div 
      class="absolute top-12 right-0 w-96 max-h-[80vh] overflow-hidden"
      transition:fly={{ y: -10, x: 10, duration: 200 }}
    >
      <Card class="shadow-xl border-2">
        <CardHeader class="pb-3">
          <div class="flex items-center justify-between">
            <CardTitle class="text-lg flex items-center gap-2">
              <Bell size={18} />
              Notifications
            </CardTitle>
            <div class="flex items-center gap-2">
              <!-- Connection Status -->
              {#if connectionStatus.connected}
                <Wifi size={16} class="text-green-600" title="Connected" />
              {:else if connectionStatus.connecting}
                <WifiOff size={16} class="text-yellow-600 animate-pulse" title="Connecting..." />
              {:else}
                <WifiOff size={16} class="text-red-600" title="Disconnected: {connectionStatus.error}" />
              {/if}
              
              <!-- Clear All Button -->
              {#if notifications.length > 0}
                <Button variant="ghost" size="sm" on:click={clearAllNotifications}>
                  <Trash2 size={14} />
                </Button>
              {/if}
              
              <!-- Close Button -->
              <Button variant="ghost" size="sm" on:click={toggleNotificationCenter}>
                <X size={16} />
              </Button>
            </div>
          </div>
        </CardHeader>

        <CardContent class="p-0">
          {#if notifications.length === 0}
            <div class="p-6 text-center text-gray-500">
              <Bell size={48} class="mx-auto mb-4 opacity-20" />
              <p class="text-sm">No notifications</p>
              <p class="text-xs mt-1">You're all caught up!</p>
            </div>
          {:else}
            <div class="max-h-96 overflow-y-auto">
              {#each sortedNotifications as notification (notification.id)}
                <div 
                  class="border-b last:border-b-0 p-4 hover:bg-gray-50 transition-colors"
                  transition:fade={{ duration: 200 }}
                >
                  <div class="flex items-start justify-between gap-3">
                    <div class="flex-1 min-w-0">
                      <!-- Notification Header -->
                      <div class="flex items-center gap-2 mb-2">
                        <span class="text-lg">
                          {getNotificationIcon(notification.type)}
                        </span>
                        <Badge variant={getNotificationTypeColor(notification.type)} class="text-xs">
                          {notification.type.replace('_', ' ')}
                        </Badge>
                        <span class="text-xs text-gray-500 ml-auto">
                          {formatTimestamp(notification.timestamp)}
                        </span>
                      </div>

                      <!-- Notification Content -->
                      <div class="text-sm text-gray-800 mb-2">
                        {#if typeof notification.data === 'string'}
                          {notification.data}
                        {:else if notification.data.message}
                          {notification.data.message}
                        {:else if notification.data.title}
                          <div class="font-medium">{notification.data.title}</div>
                          {#if notification.data.description}
                            <div class="text-gray-600 mt-1">{notification.data.description}</div>
                          {/if}
                        {:else}
                          {formatSubscriptionPayload(notification)}
                        {/if}
                      </div>

                      <!-- Metadata -->
                      {#if notification.metadata}
                        <div class="flex flex-wrap gap-1 text-xs text-gray-400">
                          {#if notification.metadata.source}
                            <span>Source: {notification.metadata.source}</span>
                          {/if}
                          {#if notification.metadata.activityId}
                            <span>Activity: {notification.metadata.activityId}</span>
                          {/if}
                          {#if notification.metadata.facultyId}
                            <span>Faculty: {notification.metadata.facultyId}</span>
                          {/if}
                        </div>
                      {/if}
                    </div>

                    <!-- Close Button -->
                    <Button 
                      variant="ghost" 
                      size="sm" 
                      on:click={() => removeNotification(notification.id)}
                      class="opacity-50 hover:opacity-100 p-1 h-6 w-6"
                    >
                      <X size={12} />
                    </Button>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </CardContent>
      </Card>
    </div>
  {/if}
</div>

<!-- Toast Notifications for High Priority -->
{#if notifications.some(n => getNotificationPriority(n.type) === 'high')}
  <div class="fixed top-4 left-1/2 transform -translate-x-1/2 z-50 space-y-2">
    {#each notifications.filter(n => getNotificationPriority(n.type) === 'high') as notification (notification.id)}
      <div 
        class="bg-red-50 border border-red-200 rounded-lg p-4 shadow-lg max-w-md"
        transition:fly={{ y: -20, duration: 300 }}
      >
        <div class="flex items-start gap-3">
          <AlertTriangle size={20} class="text-red-600 flex-shrink-0 mt-0.5" />
          <div class="flex-1">
            <div class="font-medium text-red-800 text-sm">
              {notification.type.replace('_', ' ').toUpperCase()}
            </div>
            <div class="text-red-700 text-sm mt-1">
              {typeof notification.data === 'string' ? notification.data : notification.data.message || 'High priority notification'}
            </div>
          </div>
          <Button 
            variant="ghost" 
            size="sm" 
            on:click={() => removeNotification(notification.id)}
            class="text-red-600 hover:text-red-800 p-1 h-6 w-6"
          >
            <X size={12} />
          </Button>
        </div>
      </div>
    {/each}
  </div>
{/if}

<style>
  /* Custom scrollbar for notification list */
  .max-h-96::-webkit-scrollbar {
    width: 4px;
  }
  
  .max-h-96::-webkit-scrollbar-track {
    background: #f1f1f1;
    border-radius: 2px;
  }
  
  .max-h-96::-webkit-scrollbar-thumb {
    background: #c1c1c1;
    border-radius: 2px;
  }
  
  .max-h-96::-webkit-scrollbar-thumb:hover {
    background: #a1a1a1;
  }
</style>