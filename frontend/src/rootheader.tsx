import { Outlet, useLoaderData } from "react-router";
import css from './css/rootheader.module.scss'

export async function loader() {
    const res = await fetch("/api/user/account")
    if(res.status != 200) {
        return {username: null, role: null, spaceTaken: null, space: null}
    }
    const body = await res.json()
    return {username: body.username, role: body.role, 
        freeSpace: body.space-body.spaceTaken, space: body.space}
}

function RootHeader() {
    const {username: loadedUsername, role: loadedRole, freeSpace: loadedFreeSpace, space: loadedSpace} = useLoaderData() as 
        {username: string | null, role: string | null, freeSpace: number | null, space: number | null}    

    return (
    <>
        <div className={css.header}>
            role: {loadedRole}, free space: {loadedFreeSpace}
        </div>
        <div className={css.bodyContainer}>
            <Outlet/>
        </div>
    </>
    )
}

export default RootHeader
