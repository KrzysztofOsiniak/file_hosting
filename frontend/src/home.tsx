import { useOutletContext } from 'react-router-dom'
import css from './css/home.module.scss'
import { useEffect, useRef, useState } from 'react'
import { getUnit, getUnitSize } from './util'

type Repositories = {
    repositories: {
        id: number,
        name: string,
        ownerUsername: string,
        userUploadedSpace: number
    }[]
}

type CreateRepositoryResponse = {
    id: number
}

function Home() {
    const {username, role} = useOutletContext<{username: string | null, role: string | null}>()

    return (
    <>
    <div className={css.mainShadowWrapper}>
    <div className={css.mainContainer}>
        <Repositories username={username} role={role}/>
    </div>
    </div>
    </>
    )
}

function Repositories({username, role}: {username: string | null, role: string | null}) {
    const [r, setRepositories] = useState<Repositories | null>(null)
    const [error, setError] = useState(0)

    useEffect(() => {
        if(role === null) return
        fetch("/api/repository/all-repositories")
        .then((res): Promise<Repositories> => {
            if(res.status != 200) {
                setError(res.status)
                throw new Error()
            }
            return res.json()
        })
        .then(data => {
            setRepositories(data)
        })
    }, [])

    return (
        <>
        <div className={css.repositoriesTitle}>Owned repositories</div>
        <div className={css.repositoriesContainer}>
            <OwnedRepositories username={username} role={role} r={r} error={error} setRepositories={setRepositories}/>
        </div>

        <div className={css.repositoriesTitle}>Member repositories</div>
        <div className={css.repositoriesContainer}>
            <MemberRepositories username={username} role={role} r={r} error={error}/>
        </div>
        </>
    )
}

function OwnedRepositories(
{username, role, r, error, setRepositories}: {username: string | null, role: string | null, r: Repositories | null, 
    error: number, setRepositories: React.Dispatch<React.SetStateAction<Repositories | null>>}) {
    const [loading, setLoading] = useState(false)
    const [repositoryVisibility, setRepositoryVisibility] = useState("Private")
    const [createRepositoryPopup, setCreateRepositoryPopup] = useState(false)
    const [status, setStatus] = useState("")
    const [createError, setCreateError] = useState(false)

    const repositoryName = useRef<HTMLInputElement>(null)

    if(role === null) {
        return <div className={css.informationText}>Log in to see your repositories</div>
    }
    if(error !== 0) {
        return <div className={css.informationText}>Error status {error}</div>
    }
    if(r === null) {
        return <div className={css.informationText}>Loading...</div>
    }
    if(role === "guest") {
        return <div className={css.informationText}>Guests cannot create repositories</div>
    }

    async function handleCreateRepository(e: React.MouseEvent<HTMLButtonElement>) {
        e.preventDefault()
        setLoading(true)
        const repoName = repositoryName.current!.value
        const res = await fetch('/api/repository/', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                name: repoName, visibility: repositoryVisibility === "Private" ? "private" : "public"
            })
        })
        if(res.status === 200) {
            setCreateError(false)
            setStatus("")
            setCreateRepositoryPopup(false)
            const data: CreateRepositoryResponse = await res.json()
            setRepositories((r) => {
                if(r === null) return {repositories: [{id: data.id, name: repoName, ownerUsername: username!, userUploadedSpace: 0}]}
                else return {repositories: [...r.repositories, {id: data.id, name: repoName, ownerUsername: username!, userUploadedSpace: 0}]}
            })
        }
        else {
            setCreateError(true)
        }
        if(res.status === 400) {
            setStatus("Repository name is empty or too long")
        }
        if(res.status === 409) {
            setStatus("You already have a repository with that name")
        }
        if(res.status === 500) {
            setStatus("Unknown server error")
        }
        setLoading(false)
    }
    async function handleDeleteRepository(id: number) {
        const res = await fetch(`/api/repository/${id}`, {
            method: 'DELETE',
        })
        if(res.status === 200) {
            setRepositories((r) => {
                if(r !== null) return {repositories: r.repositories.filter(repo => repo.id !== id)}
                return null
            })
        }
    }
    function handleVisibilityChange() {
        if(repositoryVisibility === "Private") setRepositoryVisibility("Public")
        else setRepositoryVisibility("Private")
    }
    function handlePopupChange() {
        if(createRepositoryPopup) setCreateRepositoryPopup(false)
        else setCreateRepositoryPopup(true)
        setStatus("")
        setCreateError(false)
    }

    return (
    <>
    {r.repositories.filter(repo => repo.ownerUsername === username).length === 0 ? 
    <button className={css.createRepositoryButton} onClick={handlePopupChange}>Create new</button> : (
    <><div className={css.ownedRepositoriesContainerElement}>
        <div className={css.ownedRepositoriesName}>Repository name</div>
        <div className={css.ownedRepositoriesUploadSize}>Your files</div>
        <div className={css.ownedRepositoriesCreateNewContainer}>
            <div className={css.ownedRepositoriesCreateNew} onClick={handlePopupChange}>
                <>Create</>
                <svg className={css.ownedRepositoriesCreateIcon} viewBox="0 -960 960 960"><path d="M440-440H200v-80h240v-240h80v240h240v80H520v240h-80v-240Z"/></svg>
            </div>
        </div>
    </div>
    {r.repositories.map(repo => repo.ownerUsername === username ? 
    <div className={`${css.ownedRepositoriesContainerElement} ${css.selectable}`} key={repo.id}>
        <div className={css.ownedRepositoriesName}>{repo.name}</div>
        <div className={css.ownedRepositoriesUploadSize}>{getUnitSize(repo.userUploadedSpace)}{getUnit(repo.userUploadedSpace)}</div>
        <div className={css.ownedRepositoriesDeleteContainer}><p onClick={() => handleDeleteRepository(repo.id)} className={css.ownedRepositoriesDelete}>Delete</p></div>
    </div>
    : <></>)}</>)
    }

    {createRepositoryPopup ?
    <div className={css.createRepositoryPopupWrapper} onClick={handlePopupChange}>
        <div className={css.createRepositoryPopup} onClick={(e) => {e.stopPropagation()}}>
            <div className={css.createRepositoryPopupHeader}>Create repository</div>
            <form className={css.formContainer}>
                <input className={css.createRepositoryInput} ref={repositoryName} type="text" placeholder="Name" autoComplete="off" maxLength={35}/>
                <div className={css.visibilityContainer}>
                    <div className={css.visibilityPick}>{repositoryVisibility}</div>
                    <svg onClick={handleVisibilityChange} className={css.switchVisibilityIcon} viewBox="0 -960 960 960"><path d="m482-200 114-113-114-113-42 42 43 43q-28 1-54.5-9T381-381q-20-20-30.5-46T340-479q0-17 4.5-34t12.5-33l-44-44q-17 25-25 53t-8 57q0 38 15 75t44 66q29 29 65 43.5t74 15.5l-38 38 42 42Zm165-170q17-25 25-53t8-57q0-38-14.5-75.5T622-622q-29-29-65.5-43T482-679l38-39-42-42-114 113 114 113 42-42-44-44q27 0 55 10.5t48 30.5q20 20 30.5 46t10.5 52q0 17-4.5 34T603-414l44 44ZM480-80q-83 0-156-31.5T197-197q-54-54-85.5-127T80-480q0-83 31.5-156T197-763q54-54 127-85.5T480-880q83 0 156 31.5T763-763q54 54 85.5 127T880-480q0 83-31.5 156T763-197q-54 54-127 85.5T480-80Zm0-80q134 0 227-93t93-227q0-134-93-227t-227-93q-134 0-227 93t-93 227q0 134 93 227t227 93Zm0-320Z"/></svg>
                </div>
                <div className={`${css.createRepositoryStatus} ${createError ? css.createRepositoryErrorStatus : ""}`}>
                    {status === "" ? "Private repository will be only visible to its members" : status}
                </div>
                <button disabled={loading} className={!loading ? css.createRepositoryButton : css.createRepositoryButtonBlocked} onClick={handleCreateRepository}>
                    Create
                </button>
            </form>
        </div>
    </div> : <></>
    }
    </>
    )
}

function MemberRepositories(
{username, role, r, error}: {username: string | null, role: string | null, r: Repositories | null, 
    error: number}) {
    if(role === null) {
        return <div className={css.informationText}>Log in to see the repositories you are a member in</div>
    }
    if(error !== 0) {
        return <div className={css.informationText}>Error status {error}</div>
    }
    if(r === null) {
        return <div className={css.informationText}>Loading...</div>
    }

    return (
        <>
        {r.repositories.filter(repo => repo.ownerUsername !== username).length === 0 ? 
        <div className={css.informationText}>No repositories</div> : 
        r.repositories.map(repo => repo.ownerUsername !== username ? <div>{repo.name}</div> : <></>)}
        </>
    )
}

export default Home
