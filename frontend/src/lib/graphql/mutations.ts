import { gql } from '@apollo/client';

// Authentication Mutations
export const LOGIN_MUTATION = gql`
  mutation Login($input: LoginInput!) {
    login(input: $input) {
      token
      user {
        id
        studentID
        email
        firstName
        lastName
        role
        isActive
        faculty {
          id
          name
          code
        }
        department {
          id
          name
          code
        }
      }
    }
  }
`;

export const REGISTER_MUTATION = gql`
  mutation Register($input: RegisterInput!) {
    register(input: $input) {
      token
      user {
        id
        studentID
        email
        firstName
        lastName
        role
        faculty {
          id
          name
          code
        }
        department {
          id
          name
          code
        }
      }
    }
  }
`;

export const REFRESH_TOKEN_MUTATION = gql`
  mutation RefreshToken {
    refreshToken {
      token
      user {
        id
        studentID
        email
        firstName
        lastName
        role
        isActive
        faculty {
          id
          name
          code
        }
        department {
          id
          name
          code
        }
      }
    }
  }
`;

// Activity Mutations
export const CREATE_ACTIVITY_MUTATION = gql`
  mutation CreateActivity($input: CreateActivityInput!) {
    createActivity(input: $input) {
      id
      title
      description
      type
      status
      startDate
      endDate
      location
      maxParticipants
      requireApproval
      points
      faculty {
        id
        name
        code
      }
      department {
        id
        name
        code
      }
      createdBy {
        id
        firstName
        lastName
      }
      createdAt
    }
  }
`;

export const UPDATE_ACTIVITY_MUTATION = gql`
  mutation UpdateActivity($id: ID!, $input: UpdateActivityInput!) {
    updateActivity(id: $id, input: $input) {
      id
      title
      description
      type
      status
      startDate
      endDate
      location
      maxParticipants
      requireApproval
      points
      faculty {
        id
        name
        code
      }
      department {
        id
        name
        code
      }
      updatedAt
    }
  }
`;

export const DELETE_ACTIVITY_MUTATION = gql`
  mutation DeleteActivity($id: ID!) {
    deleteActivity(id: $id)
  }
`;

// Participation Mutations
export const JOIN_ACTIVITY_MUTATION = gql`
  mutation JoinActivity($activityID: ID!) {
    joinActivity(activityID: $activityID) {
      id
      status
      registeredAt
      activity {
        id
        title
        type
      }
      user {
        id
        firstName
        lastName
        studentID
      }
    }
  }
`;

export const LEAVE_ACTIVITY_MUTATION = gql`
  mutation LeaveActivity($activityID: ID!) {
    leaveActivity(activityID: $activityID)
  }
`;

export const APPROVE_PARTICIPATION_MUTATION = gql`
  mutation ApproveParticipation($participationID: ID!) {
    approveParticipation(participationID: $participationID) {
      id
      status
      approvedAt
      user {
        id
        firstName
        lastName
        studentID
      }
      activity {
        id
        title
      }
    }
  }
`;

export const REJECT_PARTICIPATION_MUTATION = gql`
  mutation RejectParticipation($participationID: ID!) {
    rejectParticipation(participationID: $participationID) {
      id
      status
      user {
        id
        firstName
        lastName
        studentID
      }
      activity {
        id
        title
      }
    }
  }
`;

export const MARK_ATTENDANCE_MUTATION = gql`
  mutation MarkAttendance($participationID: ID!, $attended: Boolean!) {
    markAttendance(participationID: $participationID, attended: $attended) {
      id
      status
      attendedAt
      user {
        id
        firstName
        lastName
        studentID
      }
      activity {
        id
        title
        points
      }
    }
  }
`;

// Faculty Management Mutations (Admin only)
export const CREATE_FACULTY_MUTATION = gql`
  mutation CreateFaculty($input: CreateFacultyInput!) {
    createFaculty(input: $input) {
      id
      name
      code
      description
      isActive
      createdAt
    }
  }
`;

export const UPDATE_FACULTY_MUTATION = gql`
  mutation UpdateFaculty($id: ID!, $input: CreateFacultyInput!) {
    updateFaculty(id: $id, input: $input) {
      id
      name
      code
      description
      isActive
      updatedAt
    }
  }
`;

export const DELETE_FACULTY_MUTATION = gql`
  mutation DeleteFaculty($id: ID!) {
    deleteFaculty(id: $id)
  }
`;

// Department Management Mutations (Admin only)
export const CREATE_DEPARTMENT_MUTATION = gql`
  mutation CreateDepartment($input: CreateDepartmentInput!) {
    createDepartment(input: $input) {
      id
      name
      code
      faculty {
        id
        name
        code
      }
      isActive
      createdAt
    }
  }
`;

export const UPDATE_DEPARTMENT_MUTATION = gql`
  mutation UpdateDepartment($id: ID!, $input: CreateDepartmentInput!) {
    updateDepartment(id: $id, input: $input) {
      id
      name
      code
      faculty {
        id
        name
        code
      }
      isActive
      updatedAt
    }
  }
`;

export const DELETE_DEPARTMENT_MUTATION = gql`
  mutation DeleteDepartment($id: ID!) {
    deleteDepartment(id: $id)
  }
`;