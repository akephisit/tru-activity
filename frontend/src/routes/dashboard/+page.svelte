<script lang="ts">
	import { query } from '@apollo/client';
	import { GET_ACTIVITIES, GET_MY_PARTICIPATIONS } from '$lib/graphql/queries';
	import { user, isAdmin, isSuperAdmin, isFacultyAdmin } from '$lib/stores/auth';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Calendar, Users, Trophy, Clock } from 'lucide-svelte';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	// Auto-redirect based on user role
	onMount(() => {
		if ($isSuperAdmin) {
			goto('/dashboard/admin');
		} else if ($isFacultyAdmin) {
			goto('/dashboard/faculty-admin');
		} else if ($user?.role === 'REGULAR_ADMIN') {
			goto('/dashboard/scanner');
		}
		// Students stay on the default dashboard
	});

	const activitiesQuery = query(GET_ACTIVITIES, {
		variables: { limit: 10, status: 'ACTIVE' }
	});
	
	const myParticipationsQuery = query(GET_MY_PARTICIPATIONS);

	$: activities = $activitiesQuery.data?.activities || [];
	$: myParticipations = $myParticipationsQuery.data?.myParticipations || [];
	
	// Calculate statistics
	$: totalActivities = activities.length;
	$: myActiveParticipations = myParticipations.filter(p => 
		['PENDING', 'APPROVED'].includes(p.status)
	).length;
	$: myCompletedActivities = myParticipations.filter(p => 
		p.status === 'ATTENDED'
	).length;
	$: totalPoints = myParticipations
		.filter(p => p.status === 'ATTENDED')
		.reduce((sum, p) => sum + (p.activity.points || 0), 0);

	function formatDate(dateString: string) {
		return new Date(dateString).toLocaleDateString('th-TH', {
			year: 'numeric',
			month: 'long',
			day: 'numeric',
		});
	}

	function getStatusBadge(status: string) {
		const statusMap = {
			'DRAFT': { label: 'ร่าง', variant: 'secondary' },
			'ACTIVE': { label: 'เปิดรับสมัคร', variant: 'default' },
			'COMPLETED': { label: 'เสร็จสิ้น', variant: 'outline' },
			'CANCELLED': { label: 'ยกเลิก', variant: 'destructive' }
		};
		return statusMap[status] || { label: status, variant: 'secondary' };
	}

	function getParticipationStatusBadge(status: string) {
		const statusMap = {
			'PENDING': { label: 'รอการอนุมัติ', variant: 'secondary' },
			'APPROVED': { label: 'อนุมัติแล้ว', variant: 'default' },
			'REJECTED': { label: 'ปฏิเสธ', variant: 'destructive' },
			'ATTENDED': { label: 'เข้าร่วมแล้ว', variant: 'success' },
			'ABSENT': { label: 'ไม่เข้าร่วม', variant: 'outline' }
		};
		return statusMap[status] || { label: status, variant: 'secondary' };
	}
</script>

<div class="space-y-6">
	<!-- Welcome Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">
				ยินดีต้อนรับ, {$user?.firstName} {$user?.lastName}
			</h1>
			<p class="text-muted-foreground">
				ระบบเก็บกิจกรรมมหาวิทยาลัยเทคโนโลยีราชมงคลธัญบุรี
			</p>
		</div>
	</div>

	<!-- Statistics Cards -->
	<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
		<Card>
			<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle class="text-sm font-medium">กิจกรรมที่เปิดรับสมัคร</CardTitle>
				<Calendar class="h-4 w-4 text-muted-foreground" />
			</CardHeader>
			<CardContent>
				<div class="text-2xl font-bold">{totalActivities}</div>
				<p class="text-xs text-muted-foreground">กิจกรรมที่สามารถสมัครได้</p>
			</CardContent>
		</Card>
		
		<Card>
			<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle class="text-sm font-medium">กิจกรรมของฉัน</CardTitle>
				<Users class="h-4 w-4 text-muted-foreground" />
			</CardHeader>
			<CardContent>
				<div class="text-2xl font-bold">{myActiveParticipations}</div>
				<p class="text-xs text-muted-foreground">กิจกรรมที่ลงทะเบียนไว้</p>
			</CardContent>
		</Card>
		
		<Card>
			<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle class="text-sm font-medium">กิจกรรมที่เสร็จสิ้น</CardTitle>
				<Trophy class="h-4 w-4 text-muted-foreground" />
			</CardHeader>
			<CardContent>
				<div class="text-2xl font-bold">{myCompletedActivities}</div>
				<p class="text-xs text-muted-foreground">กิจกรรมที่เข้าร่วมแล้ว</p>
			</CardContent>
		</Card>
		
		<Card>
			<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle class="text-sm font-medium">คะแนนรวม</CardTitle>
				<Clock class="h-4 w-4 text-muted-foreground" />
			</CardHeader>
			<CardContent>
				<div class="text-2xl font-bold">{totalPoints}</div>
				<p class="text-xs text-muted-foreground">คะแนนจากกิจกรรม</p>
			</CardContent>
		</Card>
	</div>

	<div class="grid gap-6 md:grid-cols-2">
		<!-- Recent Activities -->
		<Card>
			<CardHeader>
				<CardTitle>กิจกรรมล่าสุด</CardTitle>
				<CardDescription>กิจกรรมที่เปิดรับสมัครในขณะนี้</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				{#each activities.slice(0, 5) as activity}
					<div class="flex items-start justify-between space-x-4">
						<div class="space-y-1">
							<p class="text-sm font-medium leading-none">{activity.title}</p>
							<p class="text-sm text-muted-foreground">
								{formatDate(activity.startDate)} - {formatDate(activity.endDate)}
							</p>
							{#if activity.location}
								<p class="text-xs text-muted-foreground">📍 {activity.location}</p>
							{/if}
						</div>
						<div class="text-right space-y-1">
							<Badge variant={getStatusBadge(activity.status).variant}>
								{getStatusBadge(activity.status).label}
							</Badge>
							{#if activity.points > 0}
								<p class="text-xs text-muted-foreground">{activity.points} คะแนน</p>
							{/if}
						</div>
					</div>
				{/each}
				
				{#if activities.length === 0}
					<p class="text-sm text-muted-foreground text-center py-4">
						ไม่มีกิจกรรมที่เปิดรับสมัครในขณะนี้
					</p>
				{/if}
				
				<div class="pt-4">
					<Button variant="outline" class="w-full" onclick={() => goto('/dashboard/activities')}>
						ดูกิจกรรมทั้งหมด
					</Button>
				</div>
			</CardContent>
		</Card>

		<!-- My Participations -->
		<Card>
			<CardHeader>
				<CardTitle>กิจกรรมของฉัน</CardTitle>
				<CardDescription>กิจกรรมที่คุณลงทะเบียนไว้</CardDescription>
			</CardHeader>
			<CardContent class="space-y-4">
				{#each myParticipations.slice(0, 5) as participation}
					<div class="flex items-start justify-between space-x-4">
						<div class="space-y-1">
							<p class="text-sm font-medium leading-none">{participation.activity.title}</p>
							<p class="text-sm text-muted-foreground">
								{formatDate(participation.activity.startDate)}
							</p>
							<p class="text-xs text-muted-foreground">
								ลงทะเบียนเมื่อ {formatDate(participation.registeredAt)}
							</p>
						</div>
						<div class="text-right space-y-1">
							<Badge variant={getParticipationStatusBadge(participation.status).variant}>
								{getParticipationStatusBadge(participation.status).label}
							</Badge>
							{#if participation.activity.points > 0 && participation.status === 'ATTENDED'}
								<p class="text-xs text-green-600">{participation.activity.points} คะแนน</p>
							{/if}
						</div>
					</div>
				{/each}
				
				{#if myParticipations.length === 0}
					<p class="text-sm text-muted-foreground text-center py-4">
						คุณยังไม่ได้ลงทะเบียนกิจกรรมใด ๆ
					</p>
				{/if}
				
				<div class="pt-4">
					<Button variant="outline" class="w-full" onclick={() => goto('/dashboard/my-activities')}>
						จัดการกิจกรรมของฉัน
					</Button>
				</div>
			</CardContent>
		</Card>
	</div>

	<!-- Admin Quick Actions -->
	{#if $isAdmin}
		<Card>
			<CardHeader>
				<CardTitle>การจัดการแบบด่วน</CardTitle>
				<CardDescription>เมนูสำหรับผู้ดูแลระบบ</CardDescription>
			</CardHeader>
			<CardContent>
				<div class="flex gap-2 flex-wrap">
					<Button onclick={() => goto('/dashboard/manage-activities')}>
						จัดการกิจกรรม
					</Button>
					<Button variant="outline" onclick={() => goto('/dashboard/users')}>
						จัดการผู้ใช้
					</Button>
					<Button variant="outline" onclick={() => goto('/dashboard/reports')}>
						ดูรายงาน
					</Button>
				</div>
			</CardContent>
		</Card>
	{/if}
</div>