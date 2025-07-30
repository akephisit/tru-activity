<script lang="ts">
  import { onMount } from 'svelte';
  import { client } from '$lib/graphql/client';
  import { gql } from '@apollo/client/core';
  import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
  import { Button } from '$lib/components/ui/button';
  import { Badge } from '$lib/components/ui/badge';
  import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '$lib/components/ui/dialog';
  import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '$lib/components/ui/select';
  import { Input } from '$lib/components/ui/input';
  import { Label } from '$lib/components/ui/label';
  import { 
    Plus, 
    AlertTriangle, 
    CheckCircle, 
    XCircle, 
    Calendar,
    Building2,
    Crown,
    Zap,
    Star
  } from 'lucide-svelte';

  interface Faculty {
    id: string;
    name: string;
    code: string;
  }

  interface Subscription {
    id: string;
    faculty: Faculty;
    type: string;
    status: string;
    startDate: string;
    endDate: string;
    daysUntilExpiry: number;
    needsNotification: boolean;
  }

  let subscriptions: Subscription[] = [];
  let faculties: Faculty[] = [];
  let loading = true;
  let error = '';
  let showCreateDialog = false;

  // Form data
  let formData = {
    facultyID: '',
    type: '',
    startDate: '',
    endDate: ''
  };

  const SUBSCRIPTIONS_QUERY = gql`
    query Subscriptions {
      subscriptions {
        id
        faculty {
          id
          name
          code
        }
        type
        status
        startDate
        endDate
        daysUntilExpiry
        needsNotification
      }
    }
  `;

  const FACULTIES_QUERY = gql`
    query Faculties {
      faculties {
        id
        name
        code
      }
    }
  `;

  const CREATE_SUBSCRIPTION_MUTATION = gql`
    mutation CreateSubscription($input: CreateSubscriptionInput!) {
      createSubscription(input: $input) {
        id
        faculty {
          name
          code
        }
        type
        status
        endDate
      }
    }
  `;

  onMount(async () => {
    await Promise.all([loadSubscriptions(), loadFaculties()]);
  });

  async function loadSubscriptions() {
    try {
      loading = true;
      error = '';

      const result = await client.query({
        query: SUBSCRIPTIONS_QUERY,
        fetchPolicy: 'network-only'
      });

      subscriptions = result.data.subscriptions;
    } catch (err: any) {
      error = err.message || 'Failed to load subscriptions';
      console.error('Subscriptions error:', err);
    } finally {
      loading = false;
    }
  }

  async function loadFaculties() {
    try {
      const result = await client.query({
        query: FACULTIES_QUERY,
        fetchPolicy: 'network-only'
      });

      faculties = result.data.faculties;
    } catch (err: any) {
      console.error('Faculties error:', err);
    }
  }

  async function createSubscription() {
    try {
      await client.mutate({
        mutation: CREATE_SUBSCRIPTION_MUTATION,
        variables: {
          input: {
            facultyID: formData.facultyID,
            type: formData.type,
            startDate: new Date(formData.startDate).toISOString(),
            endDate: new Date(formData.endDate).toISOString()
          }
        }
      });

      await loadSubscriptions();
      showCreateDialog = false;
      resetForm();
    } catch (err: any) {
      error = err.message || 'Failed to create subscription';
    }
  }

  function resetForm() {
    formData = {
      facultyID: '',
      type: '',
      startDate: '',
      endDate: ''
    };
  }

  function getStatusBadgeVariant(status: string) {
    switch (status.toLowerCase()) {
      case 'active': return 'default';
      case 'expired': return 'destructive';
      case 'cancelled': return 'secondary';
      default: return 'outline';
    }
  }

  function getStatusIcon(status: string) {
    switch (status.toLowerCase()) {
      case 'active': return CheckCircle;
      case 'expired': return XCircle;
      case 'cancelled': return XCircle;
      default: return AlertTriangle;
    }
  }

  function getTypeIcon(type: string) {
    switch (type.toLowerCase()) {
      case 'basic': return Calendar;
      case 'premium': return Star;
      case 'enterprise': return Crown;
      default: return Zap;
    }
  }

  function getTypeColor(type: string) {
    switch (type.toLowerCase()) {
      case 'basic': return 'text-blue-600 bg-blue-50';
      case 'premium': return 'text-purple-600 bg-purple-50';
      case 'enterprise': return 'text-amber-600 bg-amber-50';
      default: return 'text-gray-600 bg-gray-50';
    }
  }

  function getExpiryWarningClass(daysLeft: number) {
    if (daysLeft <= 1) return 'text-red-600 bg-red-50 border-red-200';
    if (daysLeft <= 7) return 'text-orange-600 bg-orange-50 border-orange-200';
    if (daysLeft <= 30) return 'text-yellow-600 bg-yellow-50 border-yellow-200';
    return 'text-green-600 bg-green-50 border-green-200';
  }

  $: expiringSubscriptions = subscriptions.filter(s => s.daysUntilExpiry <= 7 && s.status === 'ACTIVE');
  $: activeSubscriptions = subscriptions.filter(s => s.status === 'ACTIVE');
  $: expiredSubscriptions = subscriptions.filter(s => s.status === 'EXPIRED');
</script>

<div class="container mx-auto py-6 space-y-6">
  <div class="flex items-center justify-between">
    <h1 class="text-3xl font-bold">Subscription Management</h1>
    <Dialog bind:open={showCreateDialog}>
      <DialogTrigger>
        {#snippet child({ props })}
          <Button {...props}>
            <Plus size={16} class="mr-2" />
            Add Subscription
          </Button>
        {/snippet}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create New Subscription</DialogTitle>
        </DialogHeader>
        <form on:submit|preventDefault={createSubscription} class="space-y-4">
          <div>
            <Label for="faculty">Faculty</Label>
            <Select type="single" onValueChange={(value: string | string[]) => { 
              const val = Array.isArray(value) ? value[0] : value;
              if (val) formData.facultyID = val; 
            }}>
              <SelectTrigger>
                <SelectValue placeholder="Select faculty" />
              </SelectTrigger>
              <SelectContent>
                {#each faculties as faculty}
                  <SelectItem value={faculty.id}>
                    {faculty.name} ({faculty.code})
                  </SelectItem>
                {/each}
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label for="type">Subscription Type</Label>
            <Select type="single" onValueChange={(value: string | string[]) => { 
              const val = Array.isArray(value) ? value[0] : value;
              if (val) formData.type = val; 
            }}>
              <SelectTrigger>
                <SelectValue placeholder="Select subscription type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="BASIC">Basic</SelectItem>
                <SelectItem value="PREMIUM">Premium</SelectItem>
                <SelectItem value="ENTERPRISE">Enterprise</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label for="startDate">Start Date</Label>
            <Input
              id="startDate"
              type="date"
              bind:value={formData.startDate}
              required
            />
          </div>
          <div>
            <Label for="endDate">End Date</Label>
            <Input
              id="endDate"
              type="date"
              bind:value={formData.endDate}
              required
            />
          </div>
          <div class="flex justify-end gap-2">
            <Button variant="outline" type="button" onclick={() => showCreateDialog = false}>
              Cancel
            </Button>
            <Button type="submit">Create Subscription</Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  </div>

  {#if error}
    <Card class="border-red-200 bg-red-50">
      <CardContent class="pt-6">
        <div class="text-red-600">{error}</div>
      </CardContent>
    </Card>
  {/if}

  <!-- Statistics Cards -->
  <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">Active Subscriptions</CardTitle>
        <CheckCircle class="h-4 w-4 text-green-600" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold text-green-600">{activeSubscriptions.length}</div>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">Expired Subscriptions</CardTitle>
        <XCircle class="h-4 w-4 text-red-600" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold text-red-600">{expiredSubscriptions.length}</div>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">Expiring Soon</CardTitle>
        <AlertTriangle class="h-4 w-4 text-orange-600" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold text-orange-600">{expiringSubscriptions.length}</div>
      </CardContent>
    </Card>
  </div>

  <!-- Expiring Soon Alert -->
  {#if expiringSubscriptions.length > 0}
    <Card class="border-orange-200 bg-orange-50">
      <CardHeader>
        <CardTitle class="flex items-center gap-2 text-orange-700">
          <AlertTriangle size={20} />
          Subscriptions Expiring Soon
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div class="space-y-3">
          {#each expiringSubscriptions as subscription}
            <div class="flex items-center justify-between p-3 bg-white rounded border">
              <div class="flex items-center gap-3">
                <Building2 size={16} class="text-gray-600" />
                <div>
                  <span class="font-medium">{subscription.faculty.name}</span>
                  <span class="text-sm text-gray-500 ml-2">({subscription.faculty.code})</span>
                  <Badge variant="outline" class={getTypeColor(subscription.type)}>
                    {subscription.type}
                  </Badge>
                </div>
              </div>
              <div class="text-right">
                <div class="text-sm font-medium text-orange-600">
                  {subscription.daysUntilExpiry} days left
                </div>
                <div class="text-xs text-gray-500">
                  {new Date(subscription.endDate).toLocaleDateString()}
                </div>
              </div>
            </div>
          {/each}
        </div>
      </CardContent>
    </Card>
  {/if}

  <!-- All Subscriptions -->
  {#if loading}
    <div class="flex justify-center py-8">
      <div class="text-lg">Loading subscriptions...</div>
    </div>
  {:else if subscriptions.length === 0}
    <Card>
      <CardContent class="pt-6 text-center">
        <Crown size={48} class="mx-auto text-gray-400 mb-4" />
        <p class="text-lg text-gray-600">No subscriptions found</p>
        <p class="text-sm text-gray-500 mb-4">Create your first subscription to get started</p>
      </CardContent>
    </Card>
  {:else}
    <Card>
      <CardHeader>
        <CardTitle>All Subscriptions</CardTitle>
      </CardHeader>
      <CardContent>
        <div class="space-y-4">
          {#each subscriptions as subscription}
            <div class="flex items-center justify-between p-4 border rounded-lg {getExpiryWarningClass(subscription.daysUntilExpiry)}">
              <div class="flex items-center gap-4">
                <div class="p-2 rounded-full {getTypeColor(subscription.type)}">
                  <svelte:component this={getTypeIcon(subscription.type)} size={20} />
                </div>
                <div>
                  <div class="flex items-center gap-3">
                    <h3 class="font-medium">{subscription.faculty.name}</h3>
                    <Badge variant="outline" class="text-xs">
                      {subscription.faculty.code}
                    </Badge>
                  </div>
                  <div class="flex items-center gap-2 mt-1">
                    <Badge variant="outline" class={getTypeColor(subscription.type)}>
                      {subscription.type}
                    </Badge>
                    <Badge variant={getStatusBadgeVariant(subscription.status)}>
                      <svelte:component this={getStatusIcon(subscription.status)} size={12} class="mr-1" />
                      {subscription.status}
                    </Badge>
                  </div>
                </div>
              </div>
              <div class="text-right">
                <div class="text-sm font-medium">
                  {subscription.status === 'ACTIVE' ? 
                    `${subscription.daysUntilExpiry} days left` : 
                    'Expired'
                  }
                </div>
                <div class="text-xs">
                  {new Date(subscription.startDate).toLocaleDateString()} - 
                  {new Date(subscription.endDate).toLocaleDateString()}
                </div>
                {#if subscription.needsNotification}
                  <div class="text-xs flex items-center gap-1 mt-1">
                    <AlertTriangle size={12} />
                    Notification pending
                  </div>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      </CardContent>
    </Card>
  {/if}
</div>