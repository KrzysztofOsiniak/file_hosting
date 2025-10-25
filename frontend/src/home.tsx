import { useOutletContext } from 'react-router-dom'
import css from './css/home.module.scss'
import type { UserRole as UserRoleType } from './rootheader'
import { useEffect, useState } from 'react'

type Repositories = {
    repositories: {name: string}[]
}

function Home() {
    const {role} = useOutletContext<{role: UserRoleType}>()

    return (
    <>
    <div className={css.mainShadowWrapper}>
    <div className={css.mainContainer}>
        <Repositories userRole={role}/>
    </div>
    </div>
    </>
    )
}

function Repositories({userRole}: {userRole: UserRoleType}) {
    const [r, SetRepositories] = useState<Repositories | null>(null)
    const [error, setError] = useState(0)

    useEffect(() => {
        if(userRole === null) return
        fetch("/api/repository/all-repositories")
        .then((res): Promise<Repositories> => {
            if(res.status != 200) {
                setError(res.status)
                throw new Error()
            }
            return res.json()
        })
        .then(data => {
            SetRepositories(data)
        })
    }, [])

    return (
        <>
        <div className={css.repositoriesTitle}>Owned repositories</div>
        <div className={css.repositoriesContainer}>
            <OwnedRepositories userRole={userRole} r={r} error={error}/>
        </div>

        <div className={css.repositoriesTitle}>Member repositories</div>
        <div className={css.repositoriesContainer}>
            <MemberRepositories userRole={userRole} r={r} error={error}/>
        </div>
        </>
    )
}

function OwnedRepositories(
{userRole, r, error}: {userRole: UserRoleType, r: Repositories | null, error: number}) {
    if(userRole === null) {
        return <div className={css.informationText}>Log in to see owned repositories</div>
    }
    if(r === null) {
        return <div className={css.repositoriesLoading}></div>
    }
    if(error !== 0) {
        return <div className={css.informationText}>Error status {error}</div>
    }

    return (
        <>
        <div className={""}>
            {(() => {
                if(error === 0) {
                    return r.repositories.length === 0 ? "no repositories" : r.repositories.map(repo => <>{repo.name}</>)
                }
                return <>Error status: {error}</>
            })()}
        </div>
        </>
    )
}

function MemberRepositories(
    {userRole, r, error}: {userRole: UserRoleType, r: Repositories | null, error: number}) {
    if(userRole === null) {
        return <div className={css.informationText}>Log in to see the repositories you are in</div>
    }
    if(r === null) {
        return <div className={css.repositoriesLoading}></div>
    }
    if(error !== 0) {
        return <div className={css.informationText}>Error status {error}</div>
    }

    return (
        <>
        <div className={""}>
            {(() => {
                if(error === 0) {
                    return r.repositories.length === 0 ? "no repositories" : r.repositories.map(repo => <>{repo.name}</>)
                }
                return <>Error status: {error}</>
            })()}
        </div>
        </>
    )
}

export default Home
