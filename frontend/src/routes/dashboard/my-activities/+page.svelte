<script lang="ts">
  import { onMount } from 'svelte';
  import { client } from '$lib/graphql/client';
  import { gql } from '@apollo/client/core';
  import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '$lib/components/ui/select';
  import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
  import { 
    Calendar, 
    Clock, 
    MapPin, 
    Trophy, 
    Users, 
    Filter,
    Search,
    Download,
    CheckCircle,
    XCircle,
    AlertCircle,
    Hourglass
  } from 'lucide-svelte';
  import { toast } from 'svelte-sonner';

  interface Participation {
    id: string;
    status: 'PENDING' | 'APPROVED' | 'REJECTED' | 'ATTENDED' | 'ABSENT';
    registeredAt: string;
    attendedAt?: string;
    points: number;
    activity: {
      id: string;
      title: string;
      description: string;
      startDate: string;
      endDate: string;
      location?: string;
      points: number;
      status: string;
      maxParticipants: number;
      currentParticipants: number;
      faculty: {
        name: string;
        code: string;
      };
    };
  }

  interface ActivityStats {
    totalParticipations: number;
    totalPoints: number;
    attendedActivities: number;
    pendingActivities: number;
    completionRate: number;
  }

  let participations: Participation[] = [];
  let activityStats: ActivityStats = {
    totalParticipations: 0,
    totalPoints: 0,
    attendedActivities: 0,
    pendingActivities: 0,
    completionRate: 0
  };
  let loading = true;
  let error = '';
  let searchQuery = '';
  let statusFilter = 'all';
  let sortBy = 'registeredAt';
  let activeTab = 'all';

  const MY_PARTICIPATIONS_QUERY = gql`
    query MyParticipations {
      myParticipations {
        id
        status
        registeredAt
        attendedAt
        points
        activity {
          id
          title
          description
          startDate
          endDate
          location
          points
          status
          maxParticipants
          currentParticipants
          faculty {
            name
            code
          }
        }
      }
      myActivityStats {
        totalParticipations
        totalPoints
        attendedActivities
        pendingActivities
        completionRate
      }
    }
  `;

  const CANCEL_PARTICIPATION_MUTATION = gql`
    mutation CancelParticipation($participationId: ID!) {
      cancelParticipation(participationId: $participationId) {
        id
        status
      }
    }
  `;

  onMount(async () => {
    await loadParticipations();
  });

  async function loadParticipations() {
    try {
      loading = true;
      error = '';

      const result = await client.query({
        query: MY_PARTICIPATIONS_QUERY,
        fetchPolicy: 'network-only'
      });

      participations = result.data.myParticipations;
      activityStats = result.data.myActivityStats;
    } catch (err: any) {
      error = err.message || 'Failed to load participations';
      console.error('Participations error:', err);
      toast.error('ไม่สามารถโหลดข้อมูลกิจกรรมได้');
    } finally {
      loading = false;
    }
  }

  async function cancelParticipation(participationId: string) {
    if (!confirm('คุณแน่ใจหรือไม่ที่จะยกเลิกการลงทะเบียนกิจกรรมนี้?')) {
      return;
    }

    try {
      await client.mutate({
        mutation: CANCEL_PARTICIPATION_MUTATION,
        variables: { participationId }
      });

      toast.success('ยกเลิกการลงทะเบียนเรียบร้อยแล้ว');
      await loadParticipations();
    } catch (err: any) {
      toast.error('ไม่สามารถยกเลิกการลงทะเบียนได้');
      console.error('Cancel error:', err);
    }
  }

  function getStatusBadge(status: string) {
    const statusConfig = {
      'PENDING': { 
        label: 'รอการอนุมัติ', 
        variant: 'secondary' as const, 
        icon: Hourglass,
        color: 'text-yellow-600'
      },
      'APPROVED': { 
        label: 'อนุมัติแล้ว', 
        variant: 'default' as const, 
        icon: CheckCircle,
        color: 'text-blue-600'
      },
      'REJECTED': { 
        label: 'ปฏิเสธ', 
        variant: 'destructive' as const, 
        icon: XCircle,
        color: 'text-red-600'
      },
      'ATTENDED': { 
        label: 'เข้าร่วมแล้ว', 
        variant: 'default' as const, 
        icon: CheckCircle,
        color: 'text-green-600'
      },
      'ABSENT': { 
        label: 'ไม่เข้าร่วม', 
        variant: 'outline' as const, 
        icon: XCircle,
        color: 'text-gray-600'
      }
    };
    return statusConfig[status as keyof typeof statusConfig] || statusConfig['PENDING'];
  }

  function formatDate(dateString: string) {
    return new Date(dateString).toLocaleDateString('th-TH', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  function canCancelParticipation(participation: Participation) {
    return ['PENDING', 'APPROVED'].includes(participation.status) && 
           participation.activity.status === 'ACTIVE' &&
           new Date(participation.activity.startDate) > new Date();
  }

  function exportToCSV() {
    const headers = ['กิจกรรม', 'สถานะ', 'วันที่ลงทะเบียน', 'วันที่เข้าร่วม', 'คะแนน', 'คณะ'];
    const csvData = participations.map(p => [
      p.activity.title,
      getStatusBadge(p.status).label,
      formatDate(p.registeredAt),
      p.attendedAt ? formatDate(p.attendedAt) : '-',
      p.points.toString(),
      p.activity.faculty.name
    ]);

    const csvContent = [headers, ...csvData]
      .map(row => row.map(cell => `"${cell}"`).join(','))
      .join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    link.href = URL.createObjectURL(blob);
    link.download = `my-activities-${new Date().toISOString().split('T')[0]}.csv`;
    link.click();
  }

  // Filtering and sorting
  $: filteredParticipations = participations
    .filter(p => {
      // Text search
      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        if (!p.activity.title.toLowerCase().includes(query) &&
            !p.activity.description.toLowerCase().includes(query) &&
            !p.activity.faculty.name.toLowerCase().includes(query)) {
          return false;
        }
      }

      // Status filter
      if (statusFilter !== 'all' && p.status !== statusFilter) {
        return false;
      }

      // Tab filter
      if (activeTab === 'upcoming') {
        return ['PENDING', 'APPROVED'].includes(p.status) && 
               new Date(p.activity.startDate) > new Date();
      } else if (activeTab === 'completed') {
        return ['ATTENDED', 'ABSENT'].includes(p.status);
      } else if (activeTab === 'pending') {
        return p.status === 'PENDING';
      }

      return true;
    })
    .sort((a, b) => {
      switch (sortBy) {
        case 'registeredAt':
          return new Date(b.registeredAt).getTime() - new Date(a.registeredAt).getTime();
        case 'activityDate':
          return new Date(a.activity.startDate).getTime() - new Date(b.activity.startDate).getTime();
        case 'points':
          return b.points - a.points;
        case 'title':
          return a.activity.title.localeCompare(b.activity.title);
        default:
          return 0;
      }
    });
</script>

<div class="container mx-auto py-6 space-y-6">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-3xl font-bold">กิจกรรมของฉัน</h1>
      <p class="text-muted-foreground">ประวัติการเข้าร่วมกิจกรรมทั้งหมด</p>
    </div>
    <Button variant="outline" onclick={exportToCSV} disabled={participations.length === 0}>
      <Download class="w-4 h-4 mr-2" />
      Export CSV
    </Button>
  </div>

  {#if error}
    <Card class="border-red-200 bg-red-50">
      <CardContent class="pt-6">
        <div class="flex items-center gap-2 text-red-600">
          <AlertCircle size={20} />
          <span>{error}</span>
        </div>
      </CardContent>
    </Card>
  {/if}

  <!-- Statistics Cards -->
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">กิจกรรมทั้งหมด</CardTitle>
        <Calendar class="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold">{activityStats.totalParticipations}</div>
        <p class="text-xs text-muted-foreground">กิจกรรมที่ลงทะเบียน</p>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">เข้าร่วมสำเร็จ</CardTitle>
        <CheckCircle class="h-4 w-4 text-green-600" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold text-green-600">{activityStats.attendedActivities}</div>
        <p class="text-xs text-muted-foreground">กิจกรรมที่เข้าร่วมแล้ว</p>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">คะแนนรวม</CardTitle>
        <Trophy class="h-4 w-4 text-yellow-600" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold text-yellow-600">{activityStats.totalPoints}</div>
        <p class="text-xs text-muted-foreground">คะแนนสะสม</p>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">อัตราเข้าร่วม</CardTitle>
        <Users class="h-4 w-4 text-blue-600" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold text-blue-600">{activityStats.completionRate.toFixed(1)}%</div>
        <p class="text-xs text-muted-foreground">เปอร์เซ็นต์การเข้าร่วม</p>
      </CardContent>
    </Card>
  </div>

  <!-- Filters and Search -->
  <Card>
    <CardContent class="pt-6">
      <div class="flex flex-col md:flex-row gap-4">
        <div class="flex-1">
          <div class="relative">
            <Search class="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="ค้นหากิจกรรม..."
              bind:value={searchQuery}
              class="pl-10"
            />
          </div>
        </div>
        <div class="flex gap-2">
          <Select type="single" onValueChange={(value: string | string[]) => statusFilter = Array.isArray(value) ? value[0] : value}>
            <SelectTrigger class="w-40">
              <SelectValue placeholder="กรองสถานะ" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">ทั้งหมด</SelectItem>
              <SelectItem value="PENDING">รอการอนุมัติ</SelectItem>
              <SelectItem value="APPROVED">อนุมัติแล้ว</SelectItem>
              <SelectItem value="ATTENDED">เข้าร่วมแล้ว</SelectItem>
              <SelectItem value="REJECTED">ปฏิเสธ</SelectItem>
              <SelectItem value="ABSENT">ไม่เข้าร่วม</SelectItem>
            </SelectContent>
          </Select>
          <Select type="single" onValueChange={(value: string | string[]) => sortBy = Array.isArray(value) ? value[0] : value}>
            <SelectTrigger class="w-40">
              <SelectValue placeholder="เรียงตาม" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="registeredAt">วันที่ลงทะเบียน</SelectItem>
              <SelectItem value="activityDate">วันที่จัดกิจกรรม</SelectItem>
              <SelectItem value="points">คะแนน</SelectItem>
              <SelectItem value="title">ชื่อกิจกรรม</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
    </CardContent>
  </Card>

  <!-- Activity Tabs -->
  <Tabs bind:value={activeTab}>
    <TabsList class="grid w-full grid-cols-4">
      <TabsTrigger value="all">ทั้งหมด ({participations.length})</TabsTrigger>
      <TabsTrigger value="upcoming">
        กำลังจะมา ({participations.filter(p => ['PENDING', 'APPROVED'].includes(p.status) && new Date(p.activity.startDate) > new Date()).length})
      </TabsTrigger>
      <TabsTrigger value="completed">
        เสร็จสิ้น ({participations.filter(p => ['ATTENDED', 'ABSENT'].includes(p.status)).length})
      </TabsTrigger>
      <TabsTrigger value="pending">
        รออนุมัติ ({participations.filter(p => p.status === 'PENDING').length})
      </TabsTrigger>
    </TabsList>

    <TabsContent value={activeTab} class="space-y-4 mt-6">
      {#if loading}
        <div class="flex justify-center py-12">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
      {:else if filteredParticipations.length === 0}
        <Card>
          <CardContent class="pt-12 pb-12 text-center">
            <Calendar class="mx-auto h-12 w-12 text-muted-foreground mb-4" />
            <h3 class="text-lg font-medium mb-2">ไม่มีกิจกรรม</h3>
            <p class="text-muted-foreground">
              {searchQuery || statusFilter !== 'all' ? 'ไม่พบกิจกรรมตามที่ค้นหา' : 'คุณยังไม่ได้ลงทะเบียนกิจกรรมใด ๆ'}
            </p>
          </CardContent>
        </Card>
      {:else}
        {#each filteredParticipations as participation (participation.id)}
          <Card class="hover:shadow-md transition-shadow">
            <CardContent class="p-6">
              <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
                <div class="flex-1 space-y-3">
                  <div class="flex items-start justify-between">
                    <div>
                      <h3 class="text-lg font-semibold">{participation.activity.title}</h3>
                      <p class="text-sm text-muted-foreground line-clamp-2">
                        {participation.activity.description}
                      </p>
                    </div>
                    <Badge variant={getStatusBadge(participation.status).variant} class="ml-4">
                      <svelte:component 
                        this={getStatusBadge(participation.status).icon} 
                        size={14} 
                        class="mr-1"
                      />
                      {getStatusBadge(participation.status).label}
                    </Badge>
                  </div>

                  <div class="grid grid-cols-1 md:grid-cols-3 gap-3 text-sm">
                    <div class="flex items-center gap-2">
                      <Clock size={14} class="text-muted-foreground" />
                      <span>{formatDate(participation.activity.startDate)}</span>
                    </div>
                    {#if participation.activity.location}
                      <div class="flex items-center gap-2">
                        <MapPin size={14} class="text-muted-foreground" />
                        <span>{participation.activity.location}</span>
                      </div>
                    {/if}
                    <div class="flex items-center gap-2">
                      <Trophy size={14} class="text-muted-foreground" />
                      <span>{participation.points > 0 ? participation.points : participation.activity.points} คะแนน</span>
                    </div>
                  </div>

                  <div class="flex items-center gap-4 text-xs text-muted-foreground">
                    <span>คณะ: {participation.activity.faculty.name}</span>
                    <span>ลงทะเบียนเมื่อ: {formatDate(participation.registeredAt)}</span>
                    {#if participation.attendedAt}
                      <span>เข้าร่วมเมื่อ: {formatDate(participation.attendedAt)}</span>
                    {/if}
                  </div>
                </div>

                <div class="flex flex-col gap-2 min-w-fit">
                  {#if canCancelParticipation(participation)}
                    <Button 
                      variant="outline" 
                      size="sm" 
                      onclick={() => cancelParticipation(participation.id)}
                    >
                      ยกเลิก
                    </Button>
                  {/if}
                  
                  {#if participation.status === 'ATTENDED'}
                    <div class="text-center">
                      <div class="text-lg font-bold text-green-600">{participation.points}</div>
                      <div class="text-xs text-muted-foreground">คะแนนที่ได้</div>
                    </div>
                  {/if}
                </div>
              </div>
            </CardContent>
          </Card>
        {/each}
      {/if}
    </TabsContent>
  </Tabs>
</div>

<style>
  .line-clamp-2 {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
</style>