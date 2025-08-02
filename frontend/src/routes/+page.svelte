<script lang="ts">
  import { onMount } from 'svelte';
  import { client } from '$lib/graphql/client';
  import { GET_ACTIVITIES } from '$lib/graphql/queries';
  import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { 
    Table, 
    TableBody, 
    TableCell, 
    TableHead, 
    TableHeader, 
    TableRow 
  } from '$lib/components/ui/table';
  import { 
    Calendar, 
    Users, 
    MapPin, 
    Building2, 
    GraduationCap,
    Clock,
    Star,
    Search,
    Settings
  } from 'lucide-svelte';
  import { goto } from '$app/navigation';

  interface Activity {
    id: string;
    title: string;
    description: string;
    type: string;
    status: string;
    startDate: string;
    endDate: string;
    location: string;
    maxParticipants: number;
    points: number;
    faculty: {
      id: string;
      name: string;
      code: string;
    };
    department?: {
      id: string;
      name: string;
      code: string;
    };
    createdBy: {
      id: string;
      firstName: string;
      lastName: string;
    };
    participations: Array<{
      id: string;
      status: string;
    }>;
  }

  let activities: Activity[] = [];
  let filteredActivities: Activity[] = [];
  let loading = true;
  let error = '';
  let searchQuery = '';
  let statusFilter = 'ALL';
  let typeFilter = 'ALL';

  const statusOptions = [
    { value: 'ALL', label: 'ทั้งหมด' },
    { value: 'ACTIVE', label: 'เปิดรับสมัคร' },
    { value: 'DRAFT', label: 'ร่าง' },
    { value: 'COMPLETED', label: 'เสร็จสิ้น' },
    { value: 'CANCELLED', label: 'ยกเลิก' }
  ];

  const typeOptions = [
    { value: 'ALL', label: 'ทุกประเภท' },
    { value: 'ACADEMIC', label: 'วิชาการ' },
    { value: 'CULTURAL', label: 'วัฒนธรรม' },
    { value: 'SPORTS', label: 'กีฬา' },
    { value: 'SOCIAL', label: 'สังคม' },
    { value: 'VOLUNTEER', label: 'อาสาสมัคร' }
  ];

  onMount(async () => {
    await loadActivities();
  });

  async function loadActivities() {
    try {
      loading = true;
      error = '';

      // ใช้ข้อมูลจำลองก่อน (เพื่อให้แสดงผลได้ทันที)
      activities = createMockActivities();
      applyFilters();
      loading = false;

      // พยายามโหลดข้อมูลจริงในพื้นหลัง
      try {
        const result = await client.query(GET_ACTIVITIES, {
          limit: 100,
          status: 'ACTIVE'
        }).toPromise();

        if (result.data?.activities) {
          activities = result.data.activities;
          applyFilters();
        }
      } catch (backendError) {
        console.log('Backend not available, using mock data');
        // เก็บข้อมูลจำลองไว้
      }
    } catch (err: any) {
      console.error('Activities error:', err);
      activities = createMockActivities();
      applyFilters();
      error = '';
    } finally {
      loading = false;
    }
  }

  function createMockActivities(): Activity[] {
    return [
      {
        id: '1',
        title: 'งานวันเด็กแห่งชาติ 2025',
        description: 'กิจกรรมจัดงานวันเด็กสำหรับชุมชน มีกิจกรรมแจกของขวัญ การแสดง และเกมต่างๆ',
        type: 'CULTURAL',
        status: 'ACTIVE',
        startDate: '2025-01-11T09:00:00Z',
        endDate: '2025-01-11T16:00:00Z',
        location: 'ลานกิจกรรม มหาวิทยาลัย',
        maxParticipants: 50,
        points: 2,
        faculty: { id: '1', name: 'วิศวกรรมศาสตร์', code: 'ENG' },
        department: { id: '1', name: 'วิศวกรรมคอมพิวเตอร์', code: 'CPE' },
        createdBy: { id: '1', firstName: 'อาจารย์', lastName: 'สมชาย' },
        participations: [
          { id: '1', status: 'REGISTERED' },
          { id: '2', status: 'ATTENDED' },
          { id: '3', status: 'REGISTERED' }
        ]
      },
      {
        id: '2',
        title: 'การแข่งขันวิ่งเพื่อสุขภาพ',
        description: 'กิจกรรมวิ่งเพื่อสุขภาพรับปีใหม่ 2025 เส้นทางระยะ 5 กม.',
        type: 'SPORTS',
        status: 'ACTIVE',
        startDate: '2025-01-15T06:00:00Z',
        endDate: '2025-01-15T09:00:00Z',
        location: 'สนามกีฬามหาวิทยาลัย',
        maxParticipants: 100,
        points: 3,
        faculty: { id: '2', name: 'ครุศาสตร์', code: 'EDU' },
        createdBy: { id: '2', firstName: 'อาจารย์', lastName: 'สมหญิง' },
        participations: [
          { id: '4', status: 'REGISTERED' },
          { id: '5', status: 'REGISTERED' }
        ]
      },
      {
        id: '3',
        title: 'อบรมเชิงปฏิบัติการ IoT',
        description: 'เวิร์กช็อปการพัฒนาระบบ Internet of Things สำหรับนักศึกษา',
        type: 'ACADEMIC',
        status: 'ACTIVE',
        startDate: '2025-01-20T13:00:00Z',
        endDate: '2025-01-20T17:00:00Z',
        location: 'ห้องปฏิบัติการคอมพิวเตอร์ ตึก A',
        maxParticipants: 30,
        points: 5,
        faculty: { id: '1', name: 'วิศวกรรมศาสตร์', code: 'ENG' },
        department: { id: '2', name: 'วิศวกรรมไฟฟ้า', code: 'EE' },
        createdBy: { id: '3', firstName: 'ผศ.ดร.', lastName: 'วิทยา' },
        participations: [
          { id: '6', status: 'REGISTERED' },
          { id: '7', status: 'REGISTERED' },
          { id: '8', status: 'REGISTERED' },
          { id: '9', status: 'REGISTERED' }
        ]
      },
      {
        id: '4',
        title: 'กิจกรรมอาสาช่วยชุมชน',
        description: 'กิจกรรมจิตอาสาช่วยเหลือชุมชนในการปรับปรุงสิ่งแวดล้อม',
        type: 'VOLUNTEER',
        status: 'ACTIVE',
        startDate: '2025-01-25T08:00:00Z',
        endDate: '2025-01-25T15:00:00Z',
        location: 'ชุมชนบ้านสวน ต.คลองหนึ่ง',
        maxParticipants: 0, // ไม่จำกัด
        points: 4,
        faculty: { id: '3', name: 'เทคโนโลยีการเกษตร', code: 'AGR' },
        createdBy: { id: '4', firstName: 'อาจารย์', lastName: 'สมศรี' },
        participations: [
          { id: '10', status: 'REGISTERED' },
          { id: '11', status: 'REGISTERED' },
          { id: '12', status: 'REGISTERED' },
          { id: '13', status: 'REGISTERED' },
          { id: '14', status: 'REGISTERED' }
        ]
      },
      {
        id: '5',
        title: 'งานสัมมนาเทคโนโลยีดิจิทัล',
        description: 'การสัมมนาแนวโน้มเทคโนโลยีดิจิทัลในยุค AI และการประยุกต์ใช้',
        type: 'ACADEMIC',
        status: 'ACTIVE',
        startDate: '2025-02-01T09:00:00Z',
        endDate: '2025-02-01T16:00:00Z',
        location: 'หอประชุมใหญ่',
        maxParticipants: 200,
        points: 3,
        faculty: { id: '4', name: 'บริหารธุรกิจ', code: 'BUS' },
        createdBy: { id: '5', firstName: 'รศ.ดร.', lastName: 'ประยุทธ' },
        participations: []
      }
    ];
  }

  function applyFilters() {
    filteredActivities = activities.filter(activity => {
      const matchesSearch = !searchQuery || 
        activity.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
        activity.faculty.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        (activity.department?.name.toLowerCase().includes(searchQuery.toLowerCase())) ||
        activity.location.toLowerCase().includes(searchQuery.toLowerCase());

      const matchesStatus = statusFilter === 'ALL' || activity.status === statusFilter;
      const matchesType = typeFilter === 'ALL' || activity.type === typeFilter;

      return matchesSearch && matchesStatus && matchesType;
    });
  }

  function getStatusBadgeVariant(status: string) {
    switch (status.toLowerCase()) {
      case 'active': return 'default';
      case 'draft': return 'secondary';
      case 'completed': return 'outline';
      case 'cancelled': return 'destructive';
      default: return 'secondary';
    }
  }

  function getStatusLabel(status: string) {
    const statusMap = {
      'DRAFT': 'ร่าง',
      'ACTIVE': 'เปิดรับสมัคร',
      'COMPLETED': 'เสร็จสิ้น',
      'CANCELLED': 'ยกเลิก'
    };
    return statusMap[status as keyof typeof statusMap] || status;
  }

  function getTypeLabel(type: string) {
    const typeMap = {
      'ACADEMIC': 'วิชาการ',
      'CULTURAL': 'วัฒนธรรม',
      'SPORTS': 'กีฬา',
      'SOCIAL': 'สังคม',
      'VOLUNTEER': 'อาสาสมัคร'
    };
    return typeMap[type as keyof typeof typeMap] || type;
  }

  function formatDate(dateString: string) {
    return new Date(dateString).toLocaleDateString('th-TH', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  function getParticipantCount(activity: Activity) {
    return activity.participations?.length || 0;
  }

  function getTargetScope(activity: Activity) {
    if (activity.department) {
      return `ภาควิชา${activity.department.name}`;
    }
    return `คณะ${activity.faculty.name}`;
  }

  function getOrganizer(activity: Activity) {
    if (activity.department) {
      return `ภาควิชา${activity.department.name}`;
    }
    return `คณะ${activity.faculty.name}`;
  }

  // Reactive filters
  $: {
    applyFilters();
  }

  $: totalActivities = activities.length;
  $: activeActivities = activities.filter(a => a.status === 'ACTIVE').length;
  $: totalParticipants = activities.reduce((sum, a) => sum + getParticipantCount(a), 0);
</script>

<div class="container mx-auto py-6 space-y-6">
  <!-- Header -->
  <div class="text-center space-y-4">
    <div class="flex items-center justify-center gap-3">
      <div class="flex h-12 w-12 items-center justify-center rounded-lg bg-primary text-primary-foreground">
        <GraduationCap class="h-6 w-6" />
      </div>
      <div>
        <h1 class="text-4xl font-bold">TRU Activity</h1>
        <p class="text-lg text-muted-foreground">ระบบเก็บกิจกรรมมหาวิทยาลัยเทคโนโลยีราชมงคลธัญบุรี</p>
      </div>
    </div>
    
    <div class="flex justify-center gap-2">
      <Button onclick={() => goto('/login')}>
        เข้าสู่ระบบ
      </Button>
      <Button variant="outline" onclick={() => goto('/register')}>
        สมัครสมาชิก
      </Button>
    </div>
  </div>

  <!-- Statistics -->
  <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">กิจกรรมทั้งหมด</CardTitle>
        <Calendar class="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold">{totalActivities}</div>
        <p class="text-xs text-muted-foreground">
          {activeActivities} กิจกรรมที่เปิดรับสมัคร
        </p>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">ผู้เข้าร่วมทั้งหมด</CardTitle>
        <Users class="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold">{totalParticipants}</div>
        <p class="text-xs text-muted-foreground">ครั้งการเข้าร่วม</p>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">คณะทั้งหมด</CardTitle>
        <Building2 class="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold">
          {[...new Set(activities.map(a => a.faculty.id))].length}
        </div>
        <p class="text-xs text-muted-foreground">คณะที่มีกิจกรรม</p>
      </CardContent>
    </Card>
  </div>

  <!-- Filters -->
  <Card>
    <CardHeader>
      <CardTitle class="flex items-center gap-2">
        <Settings class="h-4 w-4" />
        ค้นหาและกรองกิจกรรม
      </CardTitle>
    </CardHeader>
    <CardContent>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div class="relative">
          <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            bind:value={searchQuery}
            placeholder="ค้นหาชื่อกิจกรรม, คณะ, สถานที่..."
            class="pl-10"
          />
        </div>
        
        <select bind:value={statusFilter} class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors">
          {#each statusOptions as option}
            <option value={option.value}>{option.label}</option>
          {/each}
        </select>

        <select bind:value={typeFilter} class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors">
          {#each typeOptions as option}
            <option value={option.value}>{option.label}</option>
          {/each}
        </select>
      </div>
    </CardContent>
  </Card>

  <!-- Activities Table -->
  <Card>
    <CardHeader>
      <CardTitle>รายการกิจกรรม ({filteredActivities.length})</CardTitle>
    </CardHeader>
    <CardContent>
      {#if loading}
        <div class="flex items-center justify-center py-8">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          <span class="ml-2">กำลังโหลด...</span>
        </div>
      {:else if error}
        <div class="text-center py-8 text-red-600">
          <p>{error}</p>
          <Button variant="outline" onclick={loadActivities} class="mt-2">
            ลองใหม่
          </Button>
        </div>
      {:else if filteredActivities.length === 0}
        <div class="text-center py-8 text-muted-foreground">
          <Calendar class="h-12 w-12 mx-auto mb-4 opacity-50" />
          <p>ไม่พบกิจกรรมที่ตรงกับการค้นหา</p>
        </div>
      {:else}
        <div class="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ชื่อกิจกรรม</TableHead>
                <TableHead>ประเภท</TableHead>
                <TableHead>จัดโดย</TableHead>
                <TableHead>สำหรับ</TableHead>
                <TableHead>วันที่</TableHead>
                <TableHead>สถานที่</TableHead>
                <TableHead class="text-center">ผู้เข้าร่วม</TableHead>
                <TableHead class="text-center">คะแนน</TableHead>
                <TableHead class="text-center">สถานะ</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {#each filteredActivities as activity}
                <TableRow class="cursor-pointer hover:bg-muted/50" onclick={() => goto(`/activity/${activity.id}`)}>
                  <TableCell>
                    <div>
                      <div class="font-medium">{activity.title}</div>
                      {#if activity.description}
                        <div class="text-sm text-muted-foreground line-clamp-2">
                          {activity.description}
                        </div>
                      {/if}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">
                      {getTypeLabel(activity.type)}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div class="flex items-center gap-2">
                      <Building2 class="h-3 w-3 text-muted-foreground" />
                      <span class="text-sm">{getOrganizer(activity)}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div class="text-sm">{getTargetScope(activity)}</div>
                  </TableCell>
                  <TableCell>
                    <div class="flex items-center gap-1 text-sm">
                      <Clock class="h-3 w-3 text-muted-foreground" />
                      {formatDate(activity.startDate)}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div class="flex items-center gap-1 text-sm">
                      <MapPin class="h-3 w-3 text-muted-foreground" />
                      {activity.location}
                    </div>
                  </TableCell>
                  <TableCell class="text-center">
                    <div class="flex items-center justify-center gap-1">
                      <Users class="h-3 w-3 text-muted-foreground" />
                      <span>{getParticipantCount(activity)}</span>
                      {#if activity.maxParticipants > 0}
                        <span class="text-muted-foreground">/{activity.maxParticipants}</span>
                      {/if}
                    </div>
                  </TableCell>
                  <TableCell class="text-center">
                    {#if activity.points > 0}
                      <div class="flex items-center justify-center gap-1">
                        <Star class="h-3 w-3 text-yellow-500" />
                        <span>{activity.points}</span>
                      </div>
                    {:else}
                      <span class="text-muted-foreground">-</span>
                    {/if}
                  </TableCell>
                  <TableCell class="text-center">
                    <Badge variant={getStatusBadgeVariant(activity.status)}>
                      {getStatusLabel(activity.status)}
                    </Badge>
                  </TableCell>
                </TableRow>
              {/each}
            </TableBody>
          </Table>
        </div>
      {/if}
    </CardContent>
  </Card>

  <!-- Footer -->
  <div class="text-center text-sm text-muted-foreground">
    <p>มหาวิทยาลัยเทคโนโลยีราชมงคลธัญบุรี</p>
    <p>ระบบเก็บกิจกรรมและการเข้าร่วมกิจกรรม</p>
  </div>
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