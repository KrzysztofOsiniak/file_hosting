import { useLoaderData, useOutletContext } from "react-router-dom"
import css from './css/home.module.scss'
import type { RepositoryResponse } from "./types"
import { useEffect, useState } from "react"

export default function Repository() {
    const repositoryID = useLoaderData()
    const {setHomePage} = useOutletContext<{setHomePage: React.Dispatch<React.SetStateAction<boolean>>}>()

    const [repository, setRepository] = useState<RepositoryResponse | number | null>(null)

    if(repository === 404) {
        return <>Repository not found</>
    }
    if(typeof repository === "number") {
        return <>Unknown server error</>
    }

    async function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
        if(e.target.files === null) return
        const file = e.target.files[0]
    }

    useEffect(() => {
        fetch(`/api/repository/${repositoryID}`)
        .then((res): Promise<RepositoryResponse> => {
            if(res.status != 200) {
                setRepository(res.status)
                throw new Error()
            }
            return res.json()
        })
        .then(data => {
            setRepository(data)
        })
    }, [])
    useEffect(() => setHomePage(false), [])

    if(repository === null) {
        return (
        <div className={css.mainShadowWrapper}>
        <div className={css.mainContainer}>
            loading
        </div>
        </div>
        )
    }

    return (
    <div className={css.mainShadowWrapper}>
    <div className={css.mainContainer}>
        {repository.name}
        <input type="file" onChange={handleFileChange}/>
        {repository.files.map(file => <>{file.path}</>)}
    </div>
    </div>
    )
}