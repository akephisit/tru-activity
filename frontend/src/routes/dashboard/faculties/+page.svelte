<script lang="ts">
  import { onMount } from 'svelte';
  import { client } from '$lib/graphql/client';
  import { gql } from '@apollo/client/core';
  import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { Label } from '$lib/components/ui/label';
  import { Textarea } from '$lib/components/ui/textarea';
  import { Badge } from '$lib/components/ui/badge';
  import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '$lib/components/ui/dialog';
  import { Plus, Edit2, Trash2, Building2, Users } from 'lucide-svelte';

  interface Faculty {
    id: string;
    name: string;
    code: string;
    description: string;
    isActive: boolean;
    departments: Department[];
    users: any[];
    activities: any[];
  }

  interface Department {
    id: string;
    name: string;
    code: string;
    isActive: boolean;
  }

  let faculties: Faculty[] = [];
  let loading = true;
  let error = '';
  let showCreateDialog = false;
  let showEditDialog = false;
  let editingFaculty: Faculty | null = null;

  // Form data
  let formData = {
    name: '',
    code: '',
    description: ''
  };

  const FACULTIES_QUERY = gql`
    query Faculties {
      faculties {
        id
        name
        code
        description
        isActive
        departments {
          id
          name
          code
          isActive
        }
        users {
          id
          role
        }
        activities {
          id
          status
        }
      }
    }
  `;

  const CREATE_FACULTY_MUTATION = gql`
    mutation CreateFaculty($input: CreateFacultyInput!) {
      createFaculty(input: $input) {
        id
        name
        code
        description
        isActive
      }
    }
  `;

  const UPDATE_FACULTY_MUTATION = gql`
    mutation UpdateFaculty($id: ID!, $input: CreateFacultyInput!) {
      updateFaculty(id: $id, input: $input) {
        id
        name
        code
        description
        isActive
      }
    }
  `;

  const DELETE_FACULTY_MUTATION = gql`
    mutation DeleteFaculty($id: ID!) {
      deleteFaculty(id: $id)
    }
  `;

  onMount(async () => {
    await loadFaculties();
  });

  async function loadFaculties() {
    try {
      loading = true;
      error = '';

      const result = await client.query({
        query: FACULTIES_QUERY,
        fetchPolicy: 'network-only'
      });

      faculties = result.data.faculties;
    } catch (err: any) {
      error = err.message || 'Failed to load faculties';
      console.error('Faculties error:', err);
    } finally {
      loading = false;
    }
  }

  async function createFaculty() {
    try {
      await client.mutate({
        mutation: CREATE_FACULTY_MUTATION,
        variables: {
          input: formData
        }
      });

      await loadFaculties();
      showCreateDialog = false;
      resetForm();
    } catch (err: any) {
      error = err.message || 'Failed to create faculty';
    }
  }

  async function updateFaculty() {
    if (!editingFaculty) return;

    try {
      await client.mutate({
        mutation: UPDATE_FACULTY_MUTATION,
        variables: {
          id: editingFaculty.id,
          input: formData
        }
      });

      await loadFaculties();
      showEditDialog = false;
      resetForm();
      editingFaculty = null;
    } catch (err: any) {
      error = err.message || 'Failed to update faculty';
    }
  }

  async function deleteFaculty(faculty: Faculty) {
    if (!confirm(`Are you sure you want to delete "${faculty.name}"? This action cannot be undone.`)) {
      return;
    }

    try {
      await client.mutate({
        mutation: DELETE_FACULTY_MUTATION,
        variables: {
          id: faculty.id
        }
      });

      await loadFaculties();
    } catch (err: any) {
      error = err.message || 'Failed to delete faculty';
    }
  }

  function resetForm() {
    formData = {
      name: '',
      code: '',
      description: ''
    };
  }

  function openEditDialog(faculty: Faculty) {
    editingFaculty = faculty;
    formData = {
      name: faculty.name,
      code: faculty.code,
      description: faculty.description || ''
    };
    showEditDialog = true;
  }

  function getActiveDepartmentsCount(faculty: Faculty): number {
    return faculty.departments.filter(d => d.isActive).length;
  }

  function getFacultyAdminsCount(faculty: Faculty): number {
    return faculty.users.filter(u => u.role === 'FACULTY_ADMIN').length;
  }

  function getActiveActivitiesCount(faculty: Faculty): number {
    return faculty.activities.filter(a => a.status === 'ACTIVE').length;
  }
</script>

<div class="container mx-auto py-6 space-y-6">
  <div class="flex items-center justify-between">
    <h1 class="text-3xl font-bold">Faculty Management</h1>
    <Dialog bind:open={showCreateDialog}>
      <DialogTrigger>
        {#snippet child({ props })}
          <Button {...props}>
            <Plus size={16} class="mr-2" />
            Add Faculty
          </Button>
        {/snippet}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create New Faculty</DialogTitle>
        </DialogHeader>
        <form on:submit|preventDefault={createFaculty} class="space-y-4">
          <div>
            <Label for="name">Faculty Name</Label>
            <Input
              id="name"
              bind:value={formData.name}
              placeholder="Enter faculty name"
              required
            />
          </div>
          <div>
            <Label for="code">Faculty Code</Label>
            <Input
              id="code"
              bind:value={formData.code}
              placeholder="Enter faculty code (e.g., ENG)"
              required
            />
          </div>
          <div>
            <Label for="description">Description</Label>
            <Textarea
              bind:value={formData.description}
              placeholder="Enter faculty description"
              rows={3}
              {...{ id: "description" }}
            />
          </div>
          <div class="flex justify-end gap-2">
            <Button variant="outline" type="button" onclick={() => showCreateDialog = false}>
              Cancel
            </Button>
            <Button type="submit">Create Faculty</Button>
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

  {#if loading}
    <div class="flex justify-center py-8">
      <div class="text-lg">Loading faculties...</div>
    </div>
  {:else if faculties.length === 0}
    <Card>
      <CardContent class="pt-6 text-center">
        <Building2 size={48} class="mx-auto text-gray-400 mb-4" />
        <p class="text-lg text-gray-600">No faculties found</p>
        <p class="text-sm text-gray-500 mb-4">Create your first faculty to get started</p>
      </CardContent>
    </Card>
  {:else}
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      {#each faculties as faculty}
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <div>
                <CardTitle class="flex items-center gap-2">
                  <Building2 size={20} />
                  {faculty.name}
                </CardTitle>
                <div class="flex items-center gap-2 mt-1">
                  <Badge variant="outline">{faculty.code}</Badge>
                  <Badge variant={faculty.isActive ? 'default' : 'secondary'}>
                    {faculty.isActive ? 'Active' : 'Inactive'}
                  </Badge>
                </div>
              </div>
              <div class="flex gap-2">
                <Button variant="outline" size="sm" onclick={() => openEditDialog(faculty)}>
                  <Edit2 size={16} />
                </Button>
                <Button variant="outline" size="sm" onclick={() => deleteFaculty(faculty)}>
                  <Trash2 size={16} class="text-red-600" />
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent class="space-y-4">
            {#if faculty.description}
              <p class="text-sm text-gray-600">{faculty.description}</p>
            {/if}
            
            <div class="grid grid-cols-3 gap-4 text-sm">
              <div class="text-center p-3 bg-blue-50 rounded">
                <div class="font-semibold text-blue-600">{getActiveDepartmentsCount(faculty)}</div>
                <div class="text-xs text-blue-700">Departments</div>
              </div>
              <div class="text-center p-3 bg-green-50 rounded">
                <div class="font-semibold text-green-600">{getFacultyAdminsCount(faculty)}</div>
                <div class="text-xs text-green-700">Admins</div>
              </div>
              <div class="text-center p-3 bg-purple-50 rounded">
                <div class="font-semibold text-purple-600">{getActiveActivitiesCount(faculty)}</div>
                <div class="text-xs text-purple-700">Activities</div>
              </div>
            </div>

            {#if faculty.departments.length > 0}
              <div class="space-y-2">
                <h4 class="font-medium text-sm">Departments:</h4>
                <div class="flex flex-wrap gap-1">
                  {#each faculty.departments as dept}
                    <Badge variant="outline" class="text-xs">
                      {dept.name} ({dept.code})
                    </Badge>
                  {/each}
                </div>
              </div>
            {/if}
          </CardContent>
        </Card>
      {/each}
    </div>
  {/if}

  <!-- Edit Dialog -->
  <Dialog bind:open={showEditDialog}>
    <DialogContent>
      <DialogHeader>
        <DialogTitle>Edit Faculty</DialogTitle>
      </DialogHeader>
      <form on:submit|preventDefault={updateFaculty} class="space-y-4">
        <div>
          <Label for="edit-name">Faculty Name</Label>
          <Input
            id="edit-name"
            bind:value={formData.name}
            placeholder="Enter faculty name"
            required
          />
        </div>
        <div>
          <Label for="edit-code">Faculty Code</Label>
          <Input
            id="edit-code"
            bind:value={formData.code}
            placeholder="Enter faculty code"
            required
          />
        </div>
        <div>
          <Label for="edit-description">Description</Label>
          <Textarea
            bind:value={formData.description}
            placeholder="Enter faculty description"
            rows={3}
            {...{ id: "edit-description" }}
          />
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="outline" type="button" onclick={() => showEditDialog = false}>
            Cancel
          </Button>
          <Button type="submit">Update Faculty</Button>
        </div>
      </form>
    </DialogContent>
  </Dialog>
</div>