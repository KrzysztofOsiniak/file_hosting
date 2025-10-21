import { useOutletContext } from 'react-router-dom'
import css from './css/home.module.scss'
import type { UserRole as UserRoleType } from './rootheader'
import { useEffect, useState } from 'react'

type Repositories = {
    repositories: {name: string}[]
}

function Home() {
    const { userRole } = useOutletContext<UserRoleType>()

    return (
    <>
    <div className={css.mainShadowWrapper}>
    <div className={css.mainContainer}>
        <Repositories userRole={userRole}/>
    </div>
    </div>
    </>
    )
}

function Repositories({userRole}: UserRoleType) {
    const [r, SetRepositories] = useState<Repositories | null>(null)

    useEffect(() => {
        if(userRole === null) return
        fetch("/api/repository/all-repositories")
        .then((res): Promise<Repositories> => {
            if(res.status != 200) throw new Error(res.status+"")
            return res.json()
        })
        .then(data => {
            SetRepositories(data)
        })
    }, [])

    if(userRole === null) {
        return (
            <>
            <div className={css.ownedRepositoriesContainer}>
                "log in to see or create repositories"
            </div>
            <div className={css.memberRepositoriesContainer}>
                
            </div>
            </>
        )
    }
    if(r === null) {
        return <>Loading...</>
    }

    return (
        <>
        <div className={css.ownedRepositoriesContainer}>
            {r.repositories.length === 0 ? "no repositories" : r.repositories.map(repo => <>{repo.name}</>)}
        </div>
        <div className={css.memberRepositoriesContainer}>
        </div>
        </>
    )
}

export default Home
