INSERT INTO roles (
    id,
    version,
    created_at,
    updated_at,
    created_by,
    name,
    description,
    permissions
)
VALUES (
    '028fa46b-23ba-443f-a24f-61edcde148ff',
    1,
    CURRENT_TIMESTAMP(7),
    CURRENT_TIMESTAMP(7),
    'system',
    'publisher',
    'Allows publishing Terraform modules and providers.',
    '[]'
) ON CONFLICT DO NOTHING;
