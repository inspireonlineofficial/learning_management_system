import { apiRequest } from "./client";

export type Permission = string;
export type RoleDefinition = { name: string; description?: string; permissions: Permission[] };

type BackendRole = {
  role: string;
  permissions: Array<{ resource: string; action: string }>;
};

const fromBackendRole = (role: BackendRole): RoleDefinition => ({
  name: role.role,
  permissions: role.permissions.map((p) => `${p.resource}:${p.action}`),
});

export const listRoles = () =>
  apiRequest<{ data: BackendRole[] }>("/v1/admin/rbac/roles", { auth: true }).then((result) => {
    const items = result.data.map(fromBackendRole);
    return {
      items,
      all_permissions: Array.from(new Set(items.flatMap((role) => role.permissions))).sort(),
    };
  });

export const updateRole = (name: string, permissions: Permission[]) =>
  apiRequest<BackendRole>(`/v1/admin/rbac/roles/${name}`, {
    method: "PATCH",
    auth: true,
    body: {
      permissions: permissions.map((permission) => {
        const [resource, action = "read"] = permission.split(":");
        return { resource, action };
      }),
    },
  }).then(fromBackendRole);
