import { gql } from '@apollo/client';

// User Queries
export const GET_ME = gql`
  query GetMe {
    me {
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
      createdAt
      updatedAt
    }
  }
`;

export const GET_USERS = gql`
  query GetUsers($limit: Int, $offset: Int) {
    users(limit: $limit, offset: $offset) {
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
      lastLoginAt
      createdAt
    }
  }
`;

// Faculty Queries
export const GET_FACULTIES = gql`
  query GetFaculties {
    faculties {
      id
      name
      code
      description
      isActive
      createdAt
      departments {
        id
        name
        code
      }
    }
  }
`;

export const GET_DEPARTMENTS = gql`
  query GetDepartments($facultyID: ID) {
    departments(facultyID: $facultyID) {
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

// Activity Queries
export const GET_ACTIVITIES = gql`
  query GetActivities($limit: Int, $offset: Int, $facultyID: ID, $status: ActivityStatus) {
    activities(limit: $limit, offset: $offset, facultyID: $facultyID, status: $status) {
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
      participations {
        id
        status
        user {
          id
          firstName
          lastName
          studentID
        }
      }
    }
  }
`;

export const GET_ACTIVITY = gql`
  query GetActivity($id: ID!) {
    activity(id: $id) {
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
        email
      }
      createdAt
      updatedAt
      participations {
        id
        status
        registeredAt
        approvedAt
        attendedAt
        notes
        user {
          id
          studentID
          firstName
          lastName
          email
        }
      }
    }
  }
`;

export const GET_MY_ACTIVITIES = gql`
  query GetMyActivities {
    myActivities {
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
      createdAt
      participations {
        id
        status
        user {
          id
          firstName
          lastName
        }
      }
    }
  }
`;

export const GET_MY_PARTICIPATIONS = gql`
  query GetMyParticipations {
    myParticipations {
      id
      status
      registeredAt
      approvedAt
      attendedAt
      notes
      activity {
        id
        title
        description
        type
        status
        startDate
        endDate
        location
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
      }
    }
  }
`;