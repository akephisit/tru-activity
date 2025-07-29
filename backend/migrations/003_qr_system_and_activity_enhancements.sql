-- Migration for QR system and activity management enhancements

-- Add new fields to activities table
ALTER TABLE activities 
    ADD COLUMN IF NOT EXISTS template_id INTEGER REFERENCES activity_templates(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS is_recurring BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS recurrence_rule TEXT,
    ADD COLUMN IF NOT EXISTS parent_activity_id INTEGER REFERENCES activities(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS qr_code_required BOOLEAN DEFAULT true,
    ADD COLUMN IF NOT EXISTS auto_approve BOOLEAN DEFAULT false;

-- Add new fields to participations table  
ALTER TABLE participations
    ADD COLUMN IF NOT EXISTS qr_scanned_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS scanned_by_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS scan_location VARCHAR(200);

-- Create activity_templates table
CREATE TABLE IF NOT EXISTS activity_templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    type VARCHAR(20) NOT NULL CHECK (type IN ('workshop', 'seminar', 'competition', 'volunteer', 'other')),
    default_duration INTEGER DEFAULT 60, -- in minutes
    location VARCHAR(200),
    max_participants INTEGER,
    require_approval BOOLEAN DEFAULT false,
    points INTEGER DEFAULT 0,
    qr_code_required BOOLEAN DEFAULT true,
    auto_approve BOOLEAN DEFAULT false,
    faculty_id INTEGER REFERENCES faculties(id) ON DELETE CASCADE,
    created_by_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create activity_assignments table
CREATE TABLE IF NOT EXISTS activity_assignments (
    id SERIAL PRIMARY KEY,
    activity_id INTEGER NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    admin_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_by_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    can_scan_qr BOOLEAN DEFAULT true,
    can_approve BOOLEAN DEFAULT true,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create qr_scan_logs table
CREATE TABLE IF NOT EXISTS qr_scan_logs (
    id SERIAL PRIMARY KEY,
    student_id VARCHAR(20) NOT NULL,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    activity_id INTEGER NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    scanned_by_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scan_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    qr_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    valid BOOLEAN NOT NULL,
    error_message TEXT,
    scan_location VARCHAR(200),
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_activities_template_id ON activities(template_id);
CREATE INDEX IF NOT EXISTS idx_activities_parent_activity_id ON activities(parent_activity_id);
CREATE INDEX IF NOT EXISTS idx_activities_is_recurring ON activities(is_recurring);
CREATE INDEX IF NOT EXISTS idx_activities_qr_code_required ON activities(qr_code_required);

CREATE INDEX IF NOT EXISTS idx_participations_qr_scanned_at ON participations(qr_scanned_at);
CREATE INDEX IF NOT EXISTS idx_participations_scanned_by_id ON participations(scanned_by_id);

CREATE INDEX IF NOT EXISTS idx_activity_templates_faculty_id ON activity_templates(faculty_id);
CREATE INDEX IF NOT EXISTS idx_activity_templates_created_by_id ON activity_templates(created_by_id);
CREATE INDEX IF NOT EXISTS idx_activity_templates_is_active ON activity_templates(is_active);
CREATE INDEX IF NOT EXISTS idx_activity_templates_type ON activity_templates(type);

CREATE INDEX IF NOT EXISTS idx_activity_assignments_activity_id ON activity_assignments(activity_id);
CREATE INDEX IF NOT EXISTS idx_activity_assignments_admin_id ON activity_assignments(admin_id);
CREATE INDEX IF NOT EXISTS idx_activity_assignments_assigned_by_id ON activity_assignments(assigned_by_id);

CREATE INDEX IF NOT EXISTS idx_qr_scan_logs_student_id ON qr_scan_logs(student_id);
CREATE INDEX IF NOT EXISTS idx_qr_scan_logs_user_id ON qr_scan_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_qr_scan_logs_activity_id ON qr_scan_logs(activity_id);
CREATE INDEX IF NOT EXISTS idx_qr_scan_logs_scanned_by_id ON qr_scan_logs(scanned_by_id);
CREATE INDEX IF NOT EXISTS idx_qr_scan_logs_scan_timestamp ON qr_scan_logs(scan_timestamp);
CREATE INDEX IF NOT EXISTS idx_qr_scan_logs_valid ON qr_scan_logs(valid);

-- Create unique constraints
CREATE UNIQUE INDEX IF NOT EXISTS idx_activity_assignments_unique ON activity_assignments(activity_id, admin_id) WHERE deleted_at IS NULL;

-- Add triggers for updated_at
CREATE TRIGGER update_activity_templates_updated_at BEFORE UPDATE ON activity_templates 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_activity_assignments_updated_at BEFORE UPDATE ON activity_assignments 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_qr_scan_logs_updated_at BEFORE UPDATE ON qr_scan_logs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert sample activity templates
INSERT INTO activity_templates (name, description, type, default_duration, location, require_approval, points, qr_code_required, created_by_id)
SELECT 
    'Weekly Seminar Template',
    'Template for weekly academic seminars',
    'seminar',
    120,
    'Conference Room A',
    false,
    10,
    true,
    u.id
FROM users u 
WHERE u.role IN ('super_admin', 'faculty_admin') 
LIMIT 1
ON CONFLICT DO NOTHING;

INSERT INTO activity_templates (name, description, type, default_duration, location, require_approval, points, qr_code_required, created_by_id)
SELECT 
    'Workshop Template', 
    'Template for hands-on workshops',
    'workshop',
    180,
    'Lab Room',
    true,
    20,
    true,
    u.id
FROM users u 
WHERE u.role IN ('super_admin', 'faculty_admin') 
LIMIT 1
ON CONFLICT DO NOTHING;

INSERT INTO activity_templates (name, description, type, default_duration, location, require_approval, points, qr_code_required, created_by_id)
SELECT 
    'Competition Template',
    'Template for student competitions',
    'competition',
    240,
    'Main Auditorium',
    true,
    50,
    true,
    u.id
FROM users u 
WHERE u.role IN ('super_admin', 'faculty_admin') 
LIMIT 1
ON CONFLICT DO NOTHING;

-- Update existing activities to have QR code required by default
UPDATE activities 
SET qr_code_required = true, auto_approve = false 
WHERE qr_code_required IS NULL;

-- Add constraint to ensure regular admins can only be assigned to activities
ALTER TABLE activity_assignments 
ADD CONSTRAINT check_admin_role 
CHECK (
    admin_id IN (
        SELECT id FROM users WHERE role = 'regular_admin'
    )
);

-- Add constraint to ensure recurrence rule is valid when is_recurring is true
ALTER TABLE activities 
ADD CONSTRAINT check_recurrence_rule 
CHECK (
    (is_recurring = false AND recurrence_rule IS NULL) OR 
    (is_recurring = true AND recurrence_rule IS NOT NULL AND recurrence_rule != '')
);

-- Add constraint to ensure child activities cannot have children
ALTER TABLE activities
ADD CONSTRAINT check_nested_activities
CHECK (
    (parent_activity_id IS NULL) OR 
    (parent_activity_id IS NOT NULL AND is_recurring = false)
);