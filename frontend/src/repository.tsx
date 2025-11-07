import { useLoaderData, useOutletContext } from "react-router-dom"
import css from './css/repository.module.scss'
import type { RepositoryResponse } from "./types"
import { splitFile } from "./util"
import { useEffect, useState } from "react"

type UploadStartResponse = {
    uploadParts: UploadPart[],
    fileID: number
}
type UploadPart = {
    url: string,
    part: number
}

export default function Repository() {
    const repositoryID = useLoaderData()
    const {setHomePage, setFreeSpace} = useOutletContext<{setHomePage: React.Dispatch<React.SetStateAction<boolean>>, 
        setFreeSpace: React.Dispatch<React.SetStateAction<number>>}>()

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
        const res = await fetch('/api/file/upload-start', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                key: file.name, size: file.size, repositoryID: parseInt(repositoryID)
            })
        })
        if(res.status !== 200) return
        setFreeSpace(space => space - file.size)
        let {fileID, uploadParts} = await res.json() as UploadStartResponse
        let {partCount, partSize, leftover} = splitFile(file.size)
        uploadParts = uploadParts.sort((part1, part2) => part1.part - part2.part)
        for(let i = 0, start; i < uploadParts.length; i++) {
            start = i * partSize
            if(i+1 === partCount && leftover !== 0) {
                partSize = leftover
            }
            const res = await fetch(uploadParts[i].url, {
                method: 'PUT',
                body: file.slice(start, start + partSize)
            })
            if(res.status !== 200) return
            const eTag = res.headers.get("ETag")
            if (eTag === null) return
            const res2 = await fetch('/api/file/file-part', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    fileID: fileID, eTag: eTag, part: uploadParts[i].part
                })
            })
            if(res2.status !== 200) return
        }
        const resComplete = await fetch('/api/file/upload-complete', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                id: fileID
            })
        })
        if(resComplete.status !== 200) return
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
            <div className={css.repositoryTitle}>Loading...</div>
        </div>
        </div>
        )
    }

    return (
    <div className={css.mainShadowWrapper}>
    <div className={css.mainContainer}>
        <div className={css.repositoryTitle}>{repository.name}</div>
        <div className={css.filesContainer}>
            <input type="file" onChange={handleFileChange}/>
            {repository.files.map(file => {
                if(file.type === "folder") {
                    return (
                    <div className={css.filesElement} key={file.id}>
                        {file.path}
                    </div>)
                }
                if(file.uploadDate === 0) {
                    return (
                    <div className={`${css.filesElement} ${css.inProgress}`} key={file.id}>
                        {file.path}
                    </div>)
                }
                return (
                <div className={css.filesElement} key={file.id}>
                    {file.path} {timeAgo(file.uploadDate)}
                </div>)
            })}
        </div>
    </div>
    </div>
    )
}

function timeAgo(epochSeconds: number) {
    const now = Date.now();
    const then = epochSeconds * 1000; // convert seconds to milliseconds
    const diff = now - then;

    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);
    const weeks = Math.floor(days / 7);
    const months = Math.floor(days / 30);
    const years = Math.floor(days / 365);

    if (seconds < 60) return `${seconds} seconds ago`;
    if (minutes < 60) return `${minutes} minutes ago`;
    if (hours < 24) return `${hours} hours ago`;
    if (days < 7) return `${days} days ago`;
    if (weeks < 5) return `${weeks} weeks ago`;
    if (months < 12) return `${months} months ago`;
    return `${years} years ago`;
}
