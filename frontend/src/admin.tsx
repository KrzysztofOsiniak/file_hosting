import css from './css/admin.module.scss'
import { useEffect, useRef, useState } from 'react'
import type { UserSearchResultAdmin } from './types'
import { useNavigate, useOutletContext } from 'react-router-dom'


function Admin() {
    const {setHomePage, role, username} = useOutletContext<{setHomePage: React.Dispatch<React.SetStateAction<boolean>>, 
    role: string | null, username: string}>()
    const navigate = useNavigate()

    const [userSearchResults, setUserSearchResults] = useState<UserSearchResultAdmin[]>([])
    const [warningPopup, setWarningPopup] = useState(false)
    const [warningMessage, setWarningMessage] = useState("")
    
    const userSearchRef = useRef<HTMLInputElement>(null)
    const bytesInputRefs = useRef<Record<string, HTMLInputElement | null>>({});
    

    async function handleUserSearch(e: React.MouseEvent<HTMLButtonElement, MouseEvent>) {
        e.preventDefault()
        const username = userSearchRef.current!.value
        const res = await fetch(`/api/admin/users/${username}`)
        if(res.status == 404) {
            setUserSearchResults([])
            return
        }
        if(res.status != 200) {
            setWarningMessage("Unknown error")
            setWarningPopup(true)
            return
        }
        const data = await res.json()
        setUserSearchResults(data.users)
    }

    async function setRole(userID: number, role: string) {
        const res = await fetch(`/api/admin/user/role/${userID}`, {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                role: role
            })
        })
        if(res.status != 200) {
            setWarningMessage("Unknown error")
            setWarningPopup(true)
            return
        }
        setUserSearchResults(results => results.map(u => u.id !== userID ? u : {...u, role: role}))
    }

    async function setSpace(e: React.MouseEvent<HTMLButtonElement, MouseEvent>, userID: number, bytes: number) {
        e.preventDefault()
        const res = await fetch(`/api/admin/user/storage-space`, {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                id: userID, amount: bytes
            })
        })
        if(res.status != 200) {
            setWarningMessage("Unknown error")
            setWarningPopup(true)
            return
        }
        setUserSearchResults(results => results.map(u => u.id !== userID ? u : {...u, space: bytes}))
    }

    async function removeUser(userID: number) {
        const res = await fetch(`/api/admin/user/${userID}`, {method: 'DELETE'})
        if(res.status != 200) {
            setWarningMessage("Unknown error")
            setWarningPopup(true)
            return
        }
        setUserSearchResults(results => results.filter(u => u.id !== userID))
    }

    useEffect(() => {
        if(role !== "admin") {
            navigate("/home")
        }
        setHomePage(false)
    }, [])

    return <>
    <div className={css.mainShadowWrapper}>
    <div className={css.mainContainer}>
        <div className={css.title}>Manage users</div>
        <form className={css.formContainerRow}>
            <input className={css.input} ref={userSearchRef} type="text" placeholder="Username" autoComplete="off" maxLength={100}/>
            <button className={css.searchButton} onClick={e => handleUserSearch(e)}>
                Search
            </button>
        </form>
        <div className={css.userSearchResultsContainer}>
            {userSearchResults?.map(u => {
                if(u.username === username) {
                    return <p key={u.id}></p>
                }
                return (
                <div className={css.userSearchResult} key={u.id}>
                    <p className={css.username}>{u.username}</p>
                    <form className={css.userFormContainer}>
                        <input className={css.manageInput} ref={el => {bytesInputRefs.current[u.id] = el}} type="text" placeholder={`Space: ${u.space}B`} autoComplete="off" maxLength={100}/>
                        <button className={css.setBytesButton}
                        onClick={e => {
                            e.preventDefault()
                            const id = u.id
                            if(bytesInputRefs.current[id] === null || Number.isNaN(parseInt(bytesInputRefs.current[id].value))) return
                            setSpace(e, u.id, parseInt(bytesInputRefs.current[id].value))
                            bytesInputRefs.current[id].value = ''
                        }}>
                            Set
                        </button>
                    </form>
                    <button className={`${u.role === "guest" ? css.currentRoleButton : css.roleButton}`}
                    onClick={() => {u.role === "guest" ? null : setRole(u.id, "guest")}}>Guest</button>
                    <p className={css.separator}>/</p>
                    <button className={`${u.role === "user" ? css.currentRoleButton : css.roleButton}`}
                    onClick={() => {u.role === "user" ? null : setRole(u.id, "user")}}>User</button>
                    <p className={css.separator}>/</p>
                    <button className={`${u.role === "admin" ? css.currentRoleButton : css.roleButton}`}
                    onClick={() => {u.role === "admin" ? null : setRole(u.id, "admin")}}>Admin</button>
                    <button className={css.removeUser} onClick={() => removeUser(u.id)}>Delete</button>
                </div>
                )
            })}
        </div>
    </div>
    </div>

    {warningPopup ? <WarningPopup message={warningMessage} setWarningPopup={setWarningPopup}/> : <></>}
    </>
}

function WarningPopup({message, setWarningPopup}: {message: string,setWarningPopup: React.Dispatch<React.SetStateAction<boolean>>}) {
    return (
    <div className={css.warningPopupWrapper} onClick={() => setWarningPopup(false)}>
    <div className={css.warningPopup} onClick={(e) => {e.stopPropagation()}}>
        {message}
    </div>
    </div>
    )
}

export default Admin