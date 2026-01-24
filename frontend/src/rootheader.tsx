import { Outlet, useLoaderData, Link, useNavigate } from "react-router";
import { useMemo, useState } from "react";
import css from './css/rootheader.module.scss'

import {getUnit, getUnitSize} from "./util.js"

function RootHeader() {
    const {username: loadedUsername, role: loadedRole, freeSpace: loadedFreeSpace, space: loadedSpace} = useLoaderData() as 
        {username: string | null, role: string | null, freeSpace: number | null, space: number | null}

    const navigate = useNavigate()
    const [username, setUsername] = useState(loadedUsername)
    const [role, setRole] = useState(loadedRole)
    const [profileActive, setProfileActive] = useState(false)
    const [loadingLogout, setLoadingLogout] = useState(false)
    const [homePage, setHomePage] = useState(true)
    const [freeSpace, setFreeSpace] = useState(loadedFreeSpace)

    const freeSpaceDisplay = useMemo(() => getUnitSize(freeSpace), [freeSpace])
    const freeSpaceUnit = useMemo(() => getUnit(freeSpace), [freeSpace])
    const space = useMemo(() => getUnitSize(loadedSpace), [loadedSpace])
    const spaceUnit = useMemo(() => getUnit(loadedSpace), [loadedSpace])

    async function handleLogout() {
        setLoadingLogout(true)
        const res = await fetch('/api/user/logout', {
            method: 'POST'
        })
        if(res.status != 200) {
            setLoadingLogout(false)
        }
        setLoadingLogout(false)
        setUsername(null)
        setRole(null)
        setProfileActive(false)
    }

    return (
    <>
    <div className={css.header}>
        <svg className={`${css.savedRepositories} ${username !== null ? '' : css.hidden}`} viewBox="0 -960 960 960"><path d="m354-287 126-76 126 77-33-144 111-96-146-13-58-136-58 135-146 13 111 97-33 143ZM233-120l65-281L80-590l288-25 112-265 112 265 288 25-218 189 65 281-247-149-247 149Zm247-350Z"/></svg>
        <svg onClick={() => navigate("/home")} className={`${css.home} ${!homePage ? css.homeInactive : ''}`} viewBox="0 -960 960 960"><path d="M240-200h120v-240h240v240h120v-360L480-740 240-560v360Zm-80 80v-480l320-240 320 240v480H520v-240h-80v240H160Zm320-350Z"/></svg>
        {username == null ? <Link to="/login" className={css.login}>Log in</Link> :
        <svg onClick={() => {setProfileActive(curr => !curr)}} className={`${css.profile} ${profileActive ? css.profileActive : ''}`} viewBox="0 -960 960 960"><path d="M234-276q51-39 114-61.5T480-360q69 0 132 22.5T726-276q35-41 54.5-93T800-480q0-133-93.5-226.5T480-800q-133 0-226.5 93.5T160-480q0 59 19.5 111t54.5 93Zm246-164q-59 0-99.5-40.5T340-580q0-59 40.5-99.5T480-720q59 0 99.5 40.5T620-580q0 59-40.5 99.5T480-440Zm0 360q-83 0-156-31.5T197-197q-54-54-85.5-127T80-480q0-83 31.5-156T197-763q54-54 127-85.5T480-880q83 0 156 31.5T763-763q54 54 85.5 127T880-480q0 83-31.5 156T763-197q-54 54-127 85.5T480-80Zm0-80q53 0 100-15.5t86-44.5q-39-29-86-44.5T480-280q-53 0-100 15.5T294-220q39 29 86 44.5T480-160Zm0-360q26 0 43-17t17-43q0-26-17-43t-43-17q-26 0-43 17t-17 43q0 26 17 43t43 17Zm0-60Zm0 360Z"/></svg>
        } 
    </div>
    <div className={css.bodyContainer}>
        <Outlet context={{username: username, role: role, setHomePage: setHomePage, setFreeSpace: setFreeSpace}} />
        {profileActive ?
        <div className={css.profileOptionsContainerWrapper}>
            <div className={css.profileOptionsContainer}>
                <div className={css.profileOption}>Username: {username}</div>
                <div className={css.profileOption}>Role: {role}</div>
                <div className={css.profileOption}>Free space: {freeSpaceDisplay}{freeSpaceUnit}/{space}{spaceUnit}</div>
                {role === "admin" ?
                <button className={css.logout} onClick={() => navigate("/admin")}>
                Manage users
                </button> : <></>}
                <button disabled={loadingLogout} className={!loadingLogout ? css.logout : css.logoutBlocked} onClick={handleLogout}>Log out</button>
            </div>
        </div> : <></>}
    </div>
    </>
    )
}

export default RootHeader
