<script lang="ts">
  import { onMount } from 'svelte';
  import { client } from '$lib/graphql/client';
  import { gql } from 'graphql-tag';
  import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { Button } from '$lib/components/ui/button';
  import { 
    Users, 
    Building2, 
    GraduationCap, 
    Calendar, 
    TrendingUp, 
    AlertTriangle,
    CheckCircle,
    XCircle
  } from 'lucide-svelte';

  interface SystemMetrics {
    totalFaculties: number;
    totalDepartments: number;
    totalStudents: number;
    totalActivities: number;
    totalParticipations: number;
    activeSubscriptions: number;
    expiredSubscriptions: number;
    date: string;
  }

  interface Subscription {
    id: string;
    faculty: {
      name: string;
      code: string;
    };
    type: string;
    status: string;
    daysUntilExpiry: number;
    needsNotification: boolean;
    endDate: string;
  }

  interface FacultyMetrics {
    id: string;
    faculty: {
      name: string;
    };
    totalStudents: number;
    activeStudents: number;
    totalActivities: number;
    completedActivities: number;
    averageAttendance: number;
  }

  let systemMetrics: SystemMetrics[] = [];
  let subscriptions: Subscription[] = [];
  let facultyMetrics: FacultyMetrics[] = [];
  let loading = true;
  let error = '';

  const SYSTEM_METRICS_QUERY = gql`
    query SystemMetrics($fromDate: Time, $toDate: Time) {
      systemMetrics(fromDate: $fromDate, toDate: $toDate) {
        totalFaculties
        totalDepartments
        totalStudents
        totalActivities
        totalParticipations
        activeSubscriptions
        expiredSubscriptions
        date
      }
    }
  `;

  const SUBSCRIPTIONS_QUERY = gql`
    query Subscriptions {
      subscriptions {
        id
        faculty {
          name
          code
        }
        type
        status
        daysUntilExpiry
        needsNotification
        endDate
      }
    }
  `;

  const FACULTY_METRICS_QUERY = gql`
    query FacultyMetrics($fromDate: Time, $toDate: Time) {
      facultyMetrics(fromDate: $fromDate, toDate: $toDate) {
        id
        faculty {
          name
        }
        totalStudents
        activeStudents
        totalActivities
        completedActivities
        averageAttendance
      }
    }
  `;

  onMount(async () => {
    await loadDashboardData();
  });

  async function loadDashboardData() {
    try {
      loading = true;
      error = '';

      const [systemResult, subscriptionsResult, facultyResult] = await Promise.all([
        client.query(SYSTEM_METRICS_QUERY, {
          fromDate: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
          toDate: new Date().toISOString()
        }).toPromise(),
        client.query(SUBSCRIPTIONS_QUERY, {}).toPromise(),
        client.query(FACULTY_METRICS_QUERY, {
          fromDate: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
          toDate: new Date().toISOString()
        }).toPromise()
      ]);

      systemMetrics = systemResult.data.systemMetrics;
      subscriptions = subscriptionsResult.data.subscriptions;
      facultyMetrics = facultyResult.data.facultyMetrics;
    } catch (err: any) {
      error = err.message || 'Failed to load dashboard data';
      console.error('Dashboard error:', err);
    } finally {
      loading = false;
    }
  }

  function getStatusBadgeVariant(status: string) {
    switch (status.toLowerCase()) {
      case 'active': return 'default';
      case 'expired': return 'destructive';
      case 'cancelled': return 'secondary';
      default: return 'outline';
    }
  }

  function getSubscriptionTypeColor(type: string) {
    switch (type.toLowerCase()) {
      case 'basic': return 'text-blue-600';
      case 'premium': return 'text-purple-600';
      case 'enterprise': return 'text-gold-600';
      default: return 'text-gray-600';
    }
  }

  $: latestMetrics = systemMetrics[0] || {
    totalFaculties: 0,
    totalDepartments: 0,
    totalStudents: 0,
    totalActivities: 0,
    totalParticipations: 0,
    activeSubscriptions: 0,
    expiredSubscriptions: 0
  };

  $: expiringSubscriptions = subscriptions.filter(s => s.daysUntilExpiry <= 7 && s.status === 'ACTIVE');
</script>

<div class="container mx-auto py-6 space-y-6">
  <div class="flex items-center justify-between">
    <h1 class="text-3xl font-bold">Super Admin Dashboard</h1>
    <Button onclick={loadDashboardData} disabled={loading}>
      {loading ? 'Loading...' : 'Refresh'}
    </Button>
  </div>

  {#if error}
    <Card class="border-red-200 bg-red-50">
      <CardContent class="pt-6">
        <div class="flex items-center gap-2 text-red-600">
          <XCircle size={20} />
          <span>{error}</span>
        </div>
      </CardContent>
    </Card>
  {/if}

  <!-- System Overview Cards -->
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">Total Faculties</CardTitle>
        <Building2 class="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold">{latestMetrics.totalFaculties}</div>
        <p class="text-xs text-muted-foreground">
          {latestMetrics.totalDepartments} departments
        </p>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">Total Students</CardTitle>
        <GraduationCap class="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold">{latestMetrics.totalStudents}</div>
        <p class="text-xs text-muted-foreground">Across all faculties</p>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">Total Activities</CardTitle>
        <Calendar class="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold">{latestMetrics.totalActivities}</div>
        <p class="text-xs text-muted-foreground">
          {latestMetrics.totalParticipations} participations
        </p>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">Active Subscriptions</CardTitle>
        <CheckCircle class="h-4 w-4 text-green-600" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold text-green-600">
          {latestMetrics.activeSubscriptions}
        </div>
        <p class="text-xs text-red-600">
          {latestMetrics.expiredSubscriptions} expired
        </p>
      </CardContent>
    </Card>
  </div>

  <!-- Expiring Subscriptions Alert -->
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
              <div>
                <span class="font-medium">{subscription.faculty.name}</span>
                <span class="text-sm text-gray-500 ml-2">({subscription.faculty.code})</span>
                <Badge variant="outline" class={getSubscriptionTypeColor(subscription.type)}>
                  {subscription.type}
                </Badge>
              </div>
              <div class="text-right">
                <div class="text-sm font-medium text-orange-600">
                  {subscription.daysUntilExpiry} days left
                </div>
                <div class="text-xs text-gray-500">
                  Expires: {new Date(subscription.endDate).toLocaleDateString()}
                </div>
              </div>
            </div>
          {/each}
        </div>
      </CardContent>
    </Card>
  {/if}

  <!-- Subscription Overview -->
  <Card>
    <CardHeader>
      <CardTitle>Subscription Overview</CardTitle>
    </CardHeader>
    <CardContent>
      <div class="space-y-4">
        {#each subscriptions as subscription}
          <div class="flex items-center justify-between p-4 border rounded-lg">
            <div class="flex items-center gap-4">
              <div>
                <h3 class="font-medium">{subscription.faculty.name}</h3>
                <p class="text-sm text-gray-500">{subscription.faculty.code}</p>
              </div>
              <Badge variant="outline" class={getSubscriptionTypeColor(subscription.type)}>
                {subscription.type}
              </Badge>
              <Badge variant={getStatusBadgeVariant(subscription.status)}>
                {subscription.status}
              </Badge>
            </div>
            <div class="text-right">
              <div class="text-sm">
                {subscription.status === 'ACTIVE' ? 
                  `${subscription.daysUntilExpiry} days left` : 
                  'Expired'
                }
              </div>
              <div class="text-xs text-gray-500">
                {new Date(subscription.endDate).toLocaleDateString()}
              </div>
              {#if subscription.needsNotification}
                <div class="text-xs text-orange-600 flex items-center gap-1 mt-1">
                  <AlertTriangle size={12} />
                  Needs notification
                </div>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    </CardContent>
  </Card>

  <!-- Faculty Performance -->
  <Card>
    <CardHeader>
      <CardTitle>Faculty Performance Overview</CardTitle>
    </CardHeader>
    <CardContent>
      <div class="space-y-4">
        {#each facultyMetrics as metrics}
          <div class="p-4 border rounded-lg">
            <div class="flex items-center justify-between mb-3">
              <h3 class="font-medium">{metrics.faculty.name}</h3>
              <div class="flex items-center gap-2">
                <TrendingUp size={16} class="text-green-600" />
                <span class="text-sm text-green-600">
                  {metrics.averageAttendance.toFixed(1)}% attendance
                </span>
              </div>
            </div>
            <div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
              <div>
                <span class="text-gray-500">Students:</span>
                <span class="font-medium ml-2">{metrics.totalStudents}</span>
                <span class="text-green-600 text-xs ml-1">
                  ({metrics.activeStudents} active)
                </span>
              </div>
              <div>
                <span class="text-gray-500">Activities:</span>
                <span class="font-medium ml-2">{metrics.totalActivities}</span>
                <span class="text-blue-600 text-xs ml-1">
                  ({metrics.completedActivities} completed)
                </span>
              </div>
            </div>
          </div>
        {/each}
      </div>
    </CardContent>
  </Card>
</div>