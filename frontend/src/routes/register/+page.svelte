<script lang="ts">
	import { mutation, query } from '@apollo/client';
	import { REGISTER_MUTATION } from '$lib/graphql/mutations';
	import { GET_FACULTIES } from '$lib/graphql/queries';
	import { authStore } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Select, SelectContent, SelectItem, SelectTrigger } from '$lib/components/ui/select';
	import { toast } from 'svelte-sonner';

	let studentID = $state('');
	let email = $state('');
	let firstName = $state('');
	let lastName = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let selectedFaculty = $state<string | null>(null);
	let selectedDepartment = $state<string | null>(null);
	let isLoading = $state(false);

	const register = mutation(REGISTER_MUTATION);
	const facultiesQuery = query(GET_FACULTIES);

	$: faculties = $facultiesQuery.data?.faculties || [];
	$: departments = selectedFaculty 
		? faculties.find(f => f.id === selectedFaculty)?.departments || []
		: [];

	async function handleRegister() {
		if (!studentID || !email || !firstName || !lastName || !password) {
			toast.error('กรุณากรอกข้อมูลให้ครบถ้วน');
			return;
		}

		if (password !== confirmPassword) {
			toast.error('รหัสผ่านไม่ตรงกัน');
			return;
		}

		if (password.length < 6) {
			toast.error('รหัสผ่านต้องมีอย่างน้อย 6 ตัวอักษร');
			return;
		}

		isLoading = true;
		
		try {
			const result = await register({
				variables: {
					input: {
						studentID,
						email,
						firstName,
						lastName,
						password,
						facultyID: selectedFaculty,
						departmentID: selectedDepartment
					}
				}
			});

			if (result.data?.register) {
				const { token, user } = result.data.register;
				authStore.login(token, user);
				toast.success(`สมัครสมาชิกสำเร็จ ยินดีต้อนรับ ${user.firstName} ${user.lastName}`);
				goto('/dashboard');
			}
		} catch (error: any) {
			console.error('Register error:', error);
			toast.error(error.message || 'สมัครสมาชิกไม่สำเร็จ');
		} finally {
			isLoading = false;
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
	<div class="max-w-md w-full space-y-8">
		<div>
			<h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">
				สมัครสมาชิก TRU Activity
			</h2>
			<p class="mt-2 text-center text-sm text-gray-600">
				ระบบเก็บกิจกรรมมหาวิทยาลัยเทคโนโลยีราชมงคลธัญบุรี
			</p>
		</div>
		
		<Card>
			<CardHeader>
				<CardTitle>สมัครสมาชิกใหม่</CardTitle>
				<CardDescription>กรุณากรอกข้อมูลของคุณเพื่อสมัครสมาชิก</CardDescription>
			</CardHeader>
			<CardContent>
				<form onsubmit|preventDefault={handleRegister} class="space-y-4">
					<div class="grid grid-cols-2 gap-4">
						<div>
							<Label for="firstName">ชื่อ</Label>
							<Input
								id="firstName"
								bind:value={firstName}
								placeholder="ชื่อ"
								required
								disabled={isLoading}
							/>
						</div>
						<div>
							<Label for="lastName">นามสกุล</Label>
							<Input
								id="lastName"
								bind:value={lastName}
								placeholder="นามสกุล"
								required
								disabled={isLoading}
							/>
						</div>
					</div>
					
					<div>
						<Label for="studentID">รหัสนักศึกษา</Label>
						<Input
							id="studentID"
							bind:value={studentID}
							placeholder="64xxxxxxxx"
							required
							disabled={isLoading}
						/>
					</div>
					
					<div>
						<Label for="email">อีเมล</Label>
						<Input
							id="email"
							type="email"
							bind:value={email}
							placeholder="student@mail.rmutt.ac.th"
							required
							disabled={isLoading}
						/>
					</div>

					{#if faculties.length > 0}
						<div>
							<Label for="faculty">คณะ</Label>
							<Select onValueChange={(value) => {
								selectedFaculty = value;
								selectedDepartment = null;
							}}>
								<SelectTrigger>
									{selectedFaculty 
										? faculties.find(f => f.id === selectedFaculty)?.name 
										: 'เลือกคณะ'
									}
								</SelectTrigger>
								<SelectContent>
									{#each faculties as faculty}
										<SelectItem value={faculty.id}>{faculty.name}</SelectItem>
									{/each}
								</SelectContent>
							</Select>
						</div>

						{#if departments.length > 0}
							<div>
								<Label for="department">ภาควิชา</Label>
								<Select onValueChange={(value) => selectedDepartment = value}>
									<SelectTrigger>
										{selectedDepartment 
											? departments.find(d => d.id === selectedDepartment)?.name 
											: 'เลือกภาควิชา'
										}
									</SelectTrigger>
									<SelectContent>
										{#each departments as department}
											<SelectItem value={department.id}>{department.name}</SelectItem>
										{/each}
									</SelectContent>
								</Select>
							</div>
						{/if}
					{/if}
					
					<div>
						<Label for="password">รหัสผ่าน</Label>
						<Input
							id="password"
							type="password"
							bind:value={password}
							placeholder="รหัสผ่าน (อย่างน้อย 6 ตัวอักษร)"
							required
							disabled={isLoading}
						/>
					</div>
					
					<div>
						<Label for="confirmPassword">ยืนยันรหัสผ่าน</Label>
						<Input
							id="confirmPassword"
							type="password"
							bind:value={confirmPassword}
							placeholder="ยืนยันรหัสผ่าน"
							required
							disabled={isLoading}
						/>
					</div>

					<Button type="submit" class="w-full" disabled={isLoading}>
						{isLoading ? 'กำลังสมัครสมาชิก...' : 'สมัครสมาชิก'}
					</Button>
				</form>

				<div class="mt-6">
					<div class="relative">
						<div class="absolute inset-0 flex items-center">
							<div class="w-full border-t border-gray-300" />
						</div>
						<div class="relative flex justify-center text-sm">
							<span class="px-2 bg-white text-gray-500">หรือ</span>
						</div>
					</div>

					<div class="mt-6">
						<Button variant="outline" class="w-full" onclick={() => goto('/login')}>
							เข้าสู่ระบบ
						</Button>
					</div>
				</div>
			</CardContent>
		</Card>
	</div>
</div>