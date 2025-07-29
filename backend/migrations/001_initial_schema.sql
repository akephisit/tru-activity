-- Initial database schema for TRU Activity System

-- Enable UUID extension if needed (optional)
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create enum types
CREATE TYPE user_role AS ENUM ('student', 'super_admin', 'faculty_admin', 'regular_admin');
CREATE TYPE activity_type AS ENUM ('workshop', 'seminar', 'competition', 'volunteer', 'other');
CREATE TYPE activity_status AS ENUM ('draft', 'active', 'completed', 'cancelled');
CREATE TYPE participation_status AS ENUM ('pending', 'approved', 'rejected', 'attended', 'absent');
CREATE TYPE subscription_type AS ENUM ('basic', 'premium', 'vip');
CREATE TYPE subscription_status AS ENUM ('active', 'expired', 'cancelled');

-- Create faculties table
CREATE TABLE faculties (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(10) UNIQUE NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create departments table
CREATE TABLE departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(10) NOT NULL,
    faculty_id INTEGER NOT NULL REFERENCES faculties(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(code, faculty_id)
);

-- Create users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    student_id VARCHAR(20) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    password VARCHAR(255) NOT NULL,
    role user_role DEFAULT 'student',
    qr_secret VARCHAR(32) NOT NULL,
    faculty_id INTEGER REFERENCES faculties(id) ON DELETE SET NULL,
    department_id INTEGER REFERENCES departments(id) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT true,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create activities table
CREATE TABLE activities (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    type activity_type NOT NULL,
    status activity_status DEFAULT 'draft',
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE NOT NULL,
    location VARCHAR(200),
    max_participants INTEGER,
    require_approval BOOLEAN DEFAULT false,
    points INTEGER DEFAULT 0,
    faculty_id INTEGER REFERENCES faculties(id) ON DELETE SET NULL,
    department_id INTEGER REFERENCES departments(id) ON DELETE SET NULL,
    created_by_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create participations table
CREATE TABLE participations (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_id INTEGER NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    status participation_status DEFAULT 'pending',
    registered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    approved_at TIMESTAMP WITH TIME ZONE,
    attended_at TIMESTAMP WITH TIME ZONE,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, activity_id)
);

-- Create subscriptions table
CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type subscription_type NOT NULL,
    status subscription_status DEFAULT 'active',
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for better performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_student_id ON users(student_id);
CREATE INDEX idx_users_faculty_id ON users(faculty_id);
CREATE INDEX idx_users_department_id ON users(department_id);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

CREATE INDEX idx_activities_faculty_id ON activities(faculty_id);
CREATE INDEX idx_activities_department_id ON activities(department_id);
CREATE INDEX idx_activities_created_by_id ON activities(created_by_id);
CREATE INDEX idx_activities_status ON activities(status);
CREATE INDEX idx_activities_type ON activities(type);
CREATE INDEX idx_activities_start_date ON activities(start_date);
CREATE INDEX idx_activities_deleted_at ON activities(deleted_at);

CREATE INDEX idx_participations_user_id ON participations(user_id);
CREATE INDEX idx_participations_activity_id ON participations(activity_id);
CREATE INDEX idx_participations_status ON participations(status);

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_end_date ON subscriptions(end_date);
CREATE INDEX idx_subscriptions_deleted_at ON subscriptions(deleted_at);

CREATE INDEX idx_faculties_code ON faculties(code);
CREATE INDEX idx_faculties_deleted_at ON faculties(deleted_at);

CREATE INDEX idx_departments_faculty_id ON departments(faculty_id);
CREATE INDEX idx_departments_deleted_at ON departments(deleted_at);

-- Insert sample data
INSERT INTO faculties (name, code, description) VALUES
('คณะวิศวกรรมศาสตร์', 'ENG', 'คณะวิศวกรรมศาสตร์ มหาวิทยาลัยเทคโนโลยีราชมงคลธัญบุรี'),
('คณะเทคโนโลยีสารสนเทศ', 'IT', 'คณะเทคโนโลยีสารสนเทศ มหาวิทยาลัยเทคโนโลยีราชมงคลธัญบุรี'),
('คณะบริหารธุรกิจ', 'BA', 'คณะบริหารธุรกิจ มหาวิทยาลัยเทคโนโลยีราชมงคลธัญบุรี');

INSERT INTO departments (name, code, faculty_id) VALUES
('ภาควิชาวิศวกรรมคอมพิวเตอร์', 'CPE', 1),
('ภาควิชาวิศวกรรมไฟฟ้า', 'EE', 1),
('ภาควิชาเทคโนโลยีสารสนเทศ', 'IT', 2),
('ภาควิชาวิทยาการคอมพิวเตอร์', 'CS', 2),
('ภาควิชาการจัดการ', 'MG', 3),
('ภาควิชาการตลาด', 'MK', 3);