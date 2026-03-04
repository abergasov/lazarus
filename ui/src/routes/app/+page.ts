import { redirect } from "@sveltejs/kit";
import { apiGet } from "$lib/api";

type Me = { id: string; email?: string; display_name?: string };

export async function load() {
    try {
        const me = await apiGet<Me>("/api/me");
        return { me };
    } catch {
        throw redirect(302, "/login");
    }
}