import { useLoaderData, useOutletContext } from "react-router-dom"
import css from './css/repository.module.scss'
import type { ErrorResponse, FileInProgress, RepositoryResponse } from "./types"
import { getUnit, getUnitSize, splitFile } from "./util"
import { useEffect, useRef, useState } from "react"
import { messages } from "./types"
import type { S3File, Repository } from './types'

type UploadStartResponse = {
    uploadParts: UploadPart[],
    fileID: number
}
type UploadPart = {
    url: string,
    part: number
}

export default function Repository() {
    const repositoryID = parseInt(useLoaderData())
    const {setHomePage, setFreeSpace, username} = useOutletContext<{setHomePage: React.Dispatch<React.SetStateAction<boolean>>, 
        setFreeSpace: React.Dispatch<React.SetStateAction<number>>, username: string}>()

    const [repository, setRepository] = useState<Repository | null>(null)
    const [repositoryError, setRepositoryError] = useState<null | number>(null)
    const [files, setFiles] = useState<S3File[] | null>(null)
    const [displayFiles, setDisplayFiles] = useState<S3File[] | null>(null)
    const [currentPath, setCurrentPath] = useState("")
    const [createFolderPopup, setCreateFolderPopup] = useState(false)
    const [status, setStatus] = useState("")
    const [loading, setLoading] = useState(false)
    const [warningPopup, setWarningPopup] = useState(false)
    const [warningMessage, setWarningMessage] = useState("")
    const [fileNameChangePopup, setFileNameChangePopup] = useState(false)
    const [folderNameChangePopup, setFolderNameChangePopup] = useState(false)
    const [currentlyModifiedFile, setCurrentlyModifiedFile] = useState<S3File | null>(null)
    // Currently uploaded files.
    const [filesInProgress, setFilesInProgress] = useState<FileInProgress[]>([])
    const [_, setDummyState] = useState(false)

    const nameChange = useRef<HTMLInputElement>(null)
    const folderName = useRef<HTMLInputElement>(null)
    const fileInputRef = useRef<HTMLInputElement>(null)
    const fileResumeInputRef = useRef<HTMLInputElement>(null)
    const fileResumeID = useRef(0)
    const pausedFiles = useRef<Set<number>>(new Set())

    function handleFileUpload() {
        fileInputRef.current?.click()
    }

    async function handleStartUpload(e: React.ChangeEvent<HTMLInputElement>) {
        if(e.target.files === null) return
        const file = e.target.files[0]
        const path = currentPath
        const res = await fetch('/api/file/upload-start', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                key: path + file.name, size: file.size, repositoryID: repositoryID
            })
        })
        if(res.status !== 200) {
            let message = await res.json()
            .then(body => body as ErrorResponse)
            .then(body => body.message)
            .catch(() => null)
            if(message === null) {
                setWarningMessage(`error status: ${res.status}`)
            } else if(message === messages.userHasNoSpace) {
                setWarningMessage("Not enough space to upload this file")
            } else if(message === messages.insufficientPermission) {
                setWarningMessage("You do not have the permission to upload files")
            } else if(message === messages.fileAlreadyExists) {
                setWarningMessage("A file or folder with this name already exists in this path")
            } else if(message === messages.containingFolderDoesNotExist) {
                setWarningMessage("This folder no longer exists")
            } else {
                setWarningMessage("Unknown error")
            }
            setWarningPopup(true)
            return
        }
        setFreeSpace(space => space - file.size)
        let {fileID, uploadParts} = await res.json() as UploadStartResponse
        setFiles(f => {
            if(f !== null) {
                return [...f, {
                    id: fileID,
                    ownerUsername: username,
                    path: path + file.name,
                    type: "file",
                    size: file.size,
                    uploadDate: 0
                }]
            }
            return [{
                id: fileID,
                ownerUsername: username,
                path: path + file.name,
                type: "file",
                size: file.size,
                uploadDate: 0
            }]
        })

        setFilesInProgress(f => [...f, {id: fileID, bytesUploaded: 0, bytesUploadedPrevious: 0, 
            uploadSpeedBytes: 0, timeFromLastUploadedBytes: new Date(), error: ""}])
        let {partCount, partSize, leftover} = splitFile(file.size)
        uploadParts = uploadParts.sort((part1, part2) => part1.part - part2.part)
        for(let i = 0, start; i < uploadParts.length; i++) {
            if(pausedFiles.current.has(fileID)) {
                setFilesInProgress(f => f.filter(f2 => f2.id !== fileID))
                pausedFiles.current.delete(fileID)
                return
            }
            start = i * partSize
            if(i+1 === partCount && leftover !== 0) {
                partSize = leftover
            }
            const xhr = new XMLHttpRequest()
            xhr.open('PUT', uploadParts[i].url)
            setFilesInProgress(f => f.map(f2 => {
                if(f2.id === fileID) {
                    return {...f2, bytesUploadedPrevious: 0}
                }
                return f2
            }))
            xhr.upload.addEventListener('progress', (e) => {
                if (e.lengthComputable) {
                    const bytesUploaded = e.loaded

                    setFilesInProgress(f => f.map(f2 => {
                        if(f2.id === fileID) {
                            const now = new Date()
                            const newBytesUploaded = f2.bytesUploaded + bytesUploaded - f2.bytesUploadedPrevious
                            return {...f2, bytesUploaded: newBytesUploaded, bytesUploadedPrevious: bytesUploaded, uploadSpeedBytes: (bytesUploaded - f2.bytesUploadedPrevious)/((now.getTime()-f2.timeFromLastUploadedBytes.getTime())/1000), timeFromLastUploadedBytes: new Date()}
                        }
                        return f2
                    }))
                }
            })
            let eTag = null
            xhr.send(file.slice(start, start + partSize))
            const success = await new Promise((resolve, reject) => {
                xhr.addEventListener("load", () => {
                    if (xhr.status === 200) {
                        resolve(true)
                    } else {
                        resolve(false)
                    }  
                }, { once: true })
                xhr.addEventListener("error", () => reject(null), { once: true })
            })
            .then(success => success)
            .catch(() => false)
            if(!success) {
                setFilesInProgress(f => f.filter(f2 => f2.id !== fileID))
                pausedFiles.current.delete(fileID)
                setWarningPopup(true)
                setWarningMessage(`Unknown error`)
                return
            }
            eTag = xhr.getResponseHeader('ETag')
            if (eTag === null) {
                setFilesInProgress(f => f.filter(f2 => f2.id !== fileID))
                pausedFiles.current.delete(fileID)
                return
            }

            const res2 = await fetch('/api/file/file-part', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    fileID: fileID, eTag: eTag, part: uploadParts[i].part
                })
            })
            if(res2.status !== 200) {
                if(res2.status === 403) {
                    setWarningMessage(`This file was deleted`)
                } else {
                    setWarningMessage(`Unknown error`)
                }
                setFilesInProgress(f => f.filter(f2 => f2.id !== fileID))
                pausedFiles.current.delete(fileID)
                setWarningPopup(true)
                return
            }
        }
        if(pausedFiles.current.has(fileID)) {
            setFilesInProgress(f => f.filter(f2 => f2.id !== fileID))
            pausedFiles.current.delete(fileID)
            return
        }

        // Finish the multipart upload.
        const resComplete = await fetch('/api/file/upload-complete', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                id: fileID
            })
        })
        if(resComplete.status !== 200) {
            if(resComplete.status === 403) {
                setWarningMessage(`This file was deleted`)
            } else {
                setWarningMessage(`Unknown error`)
            }
            setFilesInProgress(f => f.filter(f2 => f2.id !== fileID))
            pausedFiles.current.delete(fileID)
            setWarningPopup(true); 
            return
        }
        const completeData = await resComplete.json()
        setFiles(f => f!.map((v, _) => {
            if(v.id === fileID) {
                return {
                    ...v,
                    uploadDate: completeData.date
                }
            }
            return v
        }))
    }

    function handleFileResume(id: number) {
        fileResumeID.current = id
        fileResumeInputRef.current?.click()
    }

    async function handleResumeUpload(e: React.ChangeEvent<HTMLInputElement>) {
        if(e.target.files === null) return
        const file = e.target.files[0]
        const fileID = fileResumeID.current.valueOf()
        if(file.size !== files?.find(f => f.id === fileID)?.size) {
            setWarningMessage("The file size differs from the original file")
            setWarningPopup(true)
            return
        }
        const res = await fetch('/api/file/upload-resume', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                id: fileID
            })
        })
        if(res.status === 400) {
            setWarningMessage("This file no longer exists")
            setWarningPopup(true)
            return
        }
        if(res.status !== 200) {
            setWarningMessage("Unknown error")
            setWarningPopup(true)
            return
        }
        const { uploadParts } = await res.json() as {uploadParts: UploadPart[]}
        let {partCount, partSize, leftover} = splitFile(file.size)

        setFilesInProgress(f => {
            let alreadyUploadedBytes = file.size
            uploadParts.forEach(f => {
                if(f.part === partCount && leftover !== 0) {
                    alreadyUploadedBytes -= leftover
                } else {
                    alreadyUploadedBytes -= partSize
                }
            })
            return [...f, {id: fileID, bytesUploaded: alreadyUploadedBytes, bytesUploadedPrevious: 0, 
            uploadSpeedBytes: 0, timeFromLastUploadedBytes: new Date(), error: ""}]})
        
        for(let i = 0, start; i < uploadParts.length; i++) {
            if(pausedFiles.current.has(fileID)) {
                setFilesInProgress(f => f.filter(f2 => f2.id !== fileID))
                pausedFiles.current.delete(fileID)
                return
            }
            start = (uploadParts[i].part-1) * partSize
            let currPartSize = partSize
            if(uploadParts[i].part === partCount && leftover !== 0) {
                currPartSize = leftover
            }
            const xhr = new XMLHttpRequest()
            xhr.open('PUT', uploadParts[i].url)
            setFilesInProgress(f => f.map(f2 => {
                if(f2.id === fileID) {
                    return {...f2, bytesUploadedPrevious: 0}
                }
                return f2
            }))
            xhr.upload.addEventListener('progress', (e) => {
                if (e.lengthComputable) {
                    const bytesUploaded = e.loaded

                    setFilesInProgress(f => f.map(f2 => {
                        if(f2.id === fileID) {
                            const now = new Date()
                            const newBytesUploaded = f2.bytesUploaded + bytesUploaded - f2.bytesUploadedPrevious
                            return {...f2, bytesUploaded: newBytesUploaded, bytesUploadedPrevious: bytesUploaded, uploadSpeedBytes: (bytesUploaded - f2.bytesUploadedPrevious)/((now.getTime()-f2.timeFromLastUploadedBytes.getTime())/1000), timeFromLastUploadedBytes: new Date()}
                        }
                        return f2
                    }))
                }
            })
            let eTag = null
            xhr.send(file.slice(start, start + currPartSize))
            const success = await new Promise((resolve, reject) => {
                xhr.addEventListener("load", () => {
                    if (xhr.status === 200) {
                        resolve(true)
                    } else {
                        resolve(false)
                    }  
                }, { once: true })
                xhr.addEventListener("error", () => reject(null), { once: true })
            })
            .then(success => success)
            .catch(() => false)
            if(!success) {
                setWarningPopup(true)
                setWarningMessage(`Unknown error`)
                setFilesInProgress(f => f.filter(f2 => f2.id !== fileID))
                pausedFiles.current.delete(fileID)
                return
            }
            eTag = xhr.getResponseHeader('ETag')
            if (eTag === null) {
                setFilesInProgress(f => f.filter(f2 => f2.id !== fileID))
                pausedFiles.current.delete(fileID)
                return
            }

            const res2 = await fetch('/api/file/file-part', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    fileID: fileID, eTag: eTag, part: uploadParts[i].part
                })
            })
            if(res2.status !== 200) {
                if(res2.status === 403) {
                    setWarningMessage(`This file was deleted`)
                } else {
                    setWarningMessage(`Unknown error`)
                }
                setFilesInProgress(f => f.filter(f2 => f2.id !== fileID))
                pausedFiles.current.delete(fileID)
                setWarningPopup(true)
                return
            }
        }
        if(pausedFiles.current.has(fileID)) {
            setFilesInProgress(f => f.filter(f2 => f2.id !== fileID))
            pausedFiles.current.delete(fileID)
            return
        }

        // Finish the multipart upload.
        const resComplete = await fetch('/api/file/upload-complete', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                id: fileID
            })
        })
        if(resComplete.status !== 200) {
            if(resComplete.status === 403) {
                setWarningMessage(`This file was deleted`)
            } else {
                setWarningMessage(`Unknown error`)
            }
            setFilesInProgress(f => f.filter(f2 => f2.id !== fileID))
            pausedFiles.current.delete(fileID)
            setWarningPopup(true); 
            return
        }
        const completeData = await resComplete.json()
        setFiles(f => f!.map((v, _) => {
            if(v.id === fileID) {
                return {
                    ...v,
                    uploadDate: completeData.date
                }
            }
            return v
        }))
    }

    function handleCreateFolderClick() {
        setCreateFolderPopup(b => !b)
        setStatus("")
    }
    async function handleCreateFolder(e: React.MouseEvent<HTMLButtonElement>) {
        e.preventDefault()
        setLoading(true)
        const name = folderName.current!.value
        const path = currentPath
        const res = await fetch('/api/file/folder', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                key: currentPath + name, repositoryID: repositoryID
            })
        })
        if(res.status === 200) {
            const data = await res.json()
            handleCreateFolderClick()
            setFiles(f => {
                if(f !== null) {
                    return [...f, {
                        id: data.id, 
                        ownerUsername: username,
                        path: path + name,
                        type: "folder",
                        size: 0,
                        uploadDate: data.date
                    }]
                }
                return [{
                    id: data.id, 
                    ownerUsername: username,
                    path: path,
                    type: "folder",
                    size: 0,
                    uploadDate: data.date
                }]
            })
            setLoading(false)
            return
        }
        if(res.status === 400) {
            setStatus("Folder name is empty or too long")
            setLoading(false)
            return
        }
        if(res.status === 409) {
            setStatus("A file or folder with this name already exists in this path")
            setLoading(false)
            return
        }
        setStatus("Unknown server error")
        setLoading(false)
        return
    }

    async function handleDownload(id: number, e: React.MouseEvent<SVGSVGElement>) {
        e.stopPropagation()
        const res = await fetch(`/api/file/${id}`, {
            method: 'GET'
        })
        if(res.status !== 200) {
            setWarningMessage("Could not download this file")
            setWarningPopup(true)
            return
        }

        const body = await res.json()
        const a = document.createElement('a')
        a.href = body.url
        a.click()
        a.remove()
    }

    async function handleFileNameChange(file: S3File, e: React.MouseEvent<HTMLButtonElement, MouseEvent>) {
        e.preventDefault()
        setStatus("")
        if(loading) return
        setLoading(true)
        const newName = nameChange.current!.value
        const res = await fetch(`/api/file/name`, {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                id: file.id, name: newName
            })
        })
        if(res.status === 403) {
            setStatus("Insufficient permission to change this file")
            setLoading(false)
            return
        }
        if(res.status === 409) {
            setStatus("A file or folder with this name already exists in this path")
            setLoading(false)
            return
        }
        if(res.status === 404) {
            setStatus("This file does not exist")
            setLoading(false)
            return
        }
        if(res.status !== 200) {
            setStatus("Unknown server error")
            setLoading(false)
            return
        }
        setLoading(false)
        setFiles(f => f !== null ? f.map(f2 => {
            if(f2.id === file.id) {
                if(f2.path.includes("/")) {
                    return {...f2, path: f2.path.substring(0, f2.path.lastIndexOf('/')+1) + newName}
                } else {
                    return {...f2, path: newName}
                }
            }
            return f2
        }) : null)
        setFileNameChangePopup(false)
    }

    async function handleFolderNameChange(file: S3File, e: React.MouseEvent<HTMLButtonElement, MouseEvent>) {
        e.preventDefault()
        setStatus("")
        if(loading) return
        setLoading(true)
        const newName = nameChange.current!.value
        const res = await fetch(`/api/file/folder/name`, {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                id: file.id, name: newName
            })
        })
        if(res.status === 403) {
            setStatus("Insufficient permission to change this folder")
            setLoading(false)
            return
        }
        if(res.status === 409) {
            setStatus("A file or folder with this name already exists in this path")
            setLoading(false)
            return
        }
        if(res.status === 404) {
            setStatus("This folder does not exist")
            setLoading(false)
            return
        }
        if(res.status !== 200) {
            setStatus("Unknown server error")
            setLoading(false)
            return
        }
        setLoading(false)
        const oldPath = file.path
        setFiles(f => f !== null ? f.map(f2 => {
            if(f2.id === file.id) {
                if(f2.path.includes("/")) {
                    return {...f2, path: f2.path.substring(0, f2.path.lastIndexOf('/')+1) + newName}
                } else {
                    return {...f2, path: newName}
                }
            }
            return f2
        }) : null)
        setFiles(f => f !== null ? f.map(f2 => {
            if(f2.path.startsWith(oldPath + "/")) {
                return {...f2, path: f2.path.replace(oldPath + "/", oldPath.substring(0, oldPath.lastIndexOf('/')+1) + newName + "/")}
            }
            return f2
        }) : null)
        setFolderNameChangePopup(false)
    }

    function handlePopupClick(e: React.MouseEvent<SVGSVGElement>, file: S3File) {
        e.stopPropagation()
        if(file.type === "file") {
            setFileNameChangePopup(b => !b)
        } else {
            setFolderNameChangePopup(b => !b)
        }
        setCurrentlyModifiedFile(file)
        setStatus("")
    }

    async function handleFileDelete(id: number, e: React.MouseEvent<SVGSVGElement>) {
        e.stopPropagation()
        if(loading) return
        setLoading(true)
        const res = await fetch(`/api/file/${id}`, {
            method: 'DELETE'
        })
        if(res.status === 403) {
            setWarningMessage("Insufficient permission to delete this file")
            setWarningPopup(true)
            setLoading(false)
            return
        }
        if(res.status === 404) {
            setWarningMessage("This file does not exist")
            setWarningPopup(true)
            setLoading(false)
            return
        }
        if(res.status !== 200) {
            setWarningMessage("Unknown server error")
            setWarningPopup(true)
            setLoading(false)
            return
        }
        const size = files !== null ? files.filter(f => f.id === id)[0].size : 0
        setFreeSpace(s => s + size)
        setFiles(f => f !== null ? f.filter(f2 => f2.id !== id) : null)
        setLoading(false)
    }

    async function handleAbortUpload(id: number, e: React.MouseEvent<SVGSVGElement>) {
        e.stopPropagation()
        if(loading) return
        setLoading(true)
        const res = await fetch(`/api/file/in-progress/${id}`, {
            method: 'DELETE'
        })
        if(res.status === 403) {
            setWarningMessage("Insufficient permission to delete this file")
            setWarningPopup(true)
            setLoading(false)
            return
        }
        if(res.status === 404) {
            setWarningMessage("This file does not exist")
            setWarningPopup(true)
            setLoading(false)
            return
        }
        if(res.status !== 200) {
            setWarningMessage("Unknown server error")
            setWarningPopup(true)
            setLoading(false)
            return
        }
        const size = files !== null ? files.filter(f => f.id === id)[0].size : 0
        setFreeSpace(s => s + size)
        setFiles(f => f !== null ? f.filter(f2 => f2.id !== id) : null)
        setLoading(false)
    }

    async function handleFolderDelete(id: number, e: React.MouseEvent<SVGSVGElement>) {
        e.stopPropagation()
        if(loading) return
        setLoading(true)
        const res = await fetch(`/api/file/folder/${id}`, {
            method: 'DELETE'
        })
        if(res.status === 403) {
            setWarningMessage("Insufficient permission to delete this folder")
            setWarningPopup(true)
            setLoading(false)
            return
        }
        if(res.status === 404) {
            setWarningMessage("This folder does not exist")
            setWarningPopup(true)
            setLoading(false)
            return
        }
        if(res.status !== 200) {
            setWarningMessage("Unknown server error")
            setWarningPopup(true)
            setLoading(false)
            return
        }
        const file = displayFiles !== null ? displayFiles.filter(f => f.id === id)[0] : null
        if(file === null) {setLoading(false); return}
        const size = file.size
        const path = file.path
        setFreeSpace(s => s + size)
        setFiles(f => f !== null ? f.filter(f2 => !f2.path.startsWith(path + "/") && f2.path !== path) : null)
        setLoading(false)
    }

    async function handleVisibilityChange() {
        if(repository === null) return
        if(loading) return
        setLoading(true)
        const newVisibility = repository.visibility === "public" ? "private" : "public"
        const res = await fetch(`/api/repository/visibility`, {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                id: repositoryID, visibility: newVisibility
            })
        })
        if(res.status !== 200) {
            setWarningMessage("Unknown server error")
            setWarningPopup(true)
            setLoading(false)
            return
        }
        setRepository(r => {
            if(r === null) return r
            return {...r, visibility: newVisibility}
        })
        setLoading(false)
    }

    function handleFolderClick(path: string) {
        setCurrentPath(path)
    }

    function handleGoToPreviousFolder() {
        setCurrentPath(p => {
            let substring = p.substring(0, p.lastIndexOf('/', p.length - 2))
            if( substring === "" ) return substring
            return substring + "/"
        })
    }


    useEffect(() => {
        const interval = setInterval(() => {
            setDummyState(r => !r)
        }, 1000)
        fetch(`/api/repository/${repositoryID}`)
        .then((res): Promise<RepositoryResponse> => {
            if(res.status != 200) {
                setRepositoryError(res.status)
                throw new Error()
            }
            return res.json()
        })
        .then(data => {
            setRepository({name: data.name, userPermission: data.userPermission, ownerUsername: data.ownerUsername, visibility: data.visibility})
            setFiles(data.files)
        })
        .catch()
        return () => clearInterval(interval)
    }, [])
    useEffect(() => setHomePage(false), [])

    useEffect(() => {
        if(files === null) return
        setDisplayFiles(files.map(f => {
            if(f.type === "folder") {
                f.size = files.filter(f2 => f2.type === "file" && f2.path.startsWith(f.path + "/"))
                .map(f3 => f3.size).reduce((sum, a) => sum + a, 0)
            }
            return f
        }))
    }, [files])

    if(repositoryError === 404 || repositoryError === 401) {
        return <>Repository not found</>
    }
    if(typeof repository === "number") {
        return <>Unknown server error</>
    }

    if(displayFiles === null || repository === null) {
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
        <div className={css.repositoryTitle}>{repository.ownerUsername}<p className={css.separator}>/</p>{repository.name}</div>
        <div className={css.repositoryVisibility}>This repository is {repository.visibility}
            {repository.ownerUsername === username ? <div className={css.repositoryVisibilityChange} onClick={handleVisibilityChange}>Change visibility<svg className={css.switchVisibilityIcon} viewBox="0 -960 960 960"><path d="m482-200 114-113-114-113-42 42 43 43q-28 1-54.5-9T381-381q-20-20-30.5-46T340-479q0-17 4.5-34t12.5-33l-44-44q-17 25-25 53t-8 57q0 38 15 75t44 66q29 29 65 43.5t74 15.5l-38 38 42 42Zm165-170q17-25 25-53t8-57q0-38-14.5-75.5T622-622q-29-29-65.5-43T482-679l38-39-42-42-114 113 114 113 42-42-44-44q27 0 55 10.5t48 30.5q20 20 30.5 46t10.5 52q0 17-4.5 34T603-414l44 44ZM480-80q-83 0-156-31.5T197-197q-54-54-85.5-127T80-480q0-83 31.5-156T197-763q54-54 127-85.5T480-880q83 0 156 31.5T763-763q54 54 85.5 127T880-480q0 83-31.5 156T763-197q-54 54-127 85.5T480-80Zm0-80q134 0 227-93t93-227q0-134-93-227t-227-93q-134 0-227 93t-93 227q0 134 93 227t227 93Zm0-320Z"/></svg></div>
            : <></>}
        </div>
        <div className={css.filesContainer}>
            <input type="file" onChange={handleStartUpload} ref={fileInputRef} style={{display: 'none'}}/>
            <input type="file" onChange={handleResumeUpload} ref={fileResumeInputRef} style={{display: 'none'}}/>
            <div className={css.filesElement}>
                <div className={css.currentPath}>{"/" + currentPath}</div>
                <div className={css.uploadContainer} onClick={handleFileUpload}>
                    Upload file
                    <svg viewBox="0 -960 960 960" className={css.uploadIcon}><path d="M440-200h80v-167l64 64 56-57-160-160-160 160 57 56 63-63v167ZM240-80q-33 0-56.5-23.5T160-160v-640q0-33 23.5-56.5T240-880h320l240 240v480q0 33-23.5 56.5T720-80H240Zm280-520v-200H240v640h480v-440H520ZM240-800v200-200 640-640Z"/></svg>
                </div>
                <div className={css.uploadContainer} onClick={handleCreateFolderClick}>
                    Create folder
                    <svg viewBox="0 -960 960 960" className={css.createFolderIcon}><path d="M560-320h80v-80h80v-80h-80v-80h-80v80h-80v80h80v80ZM160-160q-33 0-56.5-23.5T80-240v-480q0-33 23.5-56.5T160-800h240l80 80h320q33 0 56.5 23.5T880-640v400q0 33-23.5 56.5T800-160H160Zm0-80h640v-400H447l-80-80H160v480Zm0 0v-480 480Z"/></svg>
                </div>
            </div>
            {currentPath !== "" ?
                <div onClick={handleGoToPreviousFolder} className={`${css.filesElement} ${css.selectable}`}>
                    <div className={`${css.fileName} ${css.folderElement}`} title={"go back"}>. .</div>
                </div>
            : <></>}
            {displayFiles.filter(file => {
                if (!file.path.startsWith(currentPath)) return false
                    const remainder = file.path.slice(currentPath.length)
                    return !remainder.includes('/')
                }).sort((a, b) => a.path.split('/').pop()!.localeCompare(b.path.split('/').pop()!)).map(file => {
                if(file.type === "folder") {
                    return (
                    <div onClick={() => handleFolderClick(file.path + "/")} className={`${css.filesElement} ${css.selectable}`} key={file.id}>
                        <div className={`${css.fileName} ${css.folderElement}`} title={file.path}>{file.path.split('/').pop()}</div>
                        <div className={css.username}>{file.ownerUsername}</div>
                        <div className={css.size}>{getUnitSize(file.size)}{getUnit(file.size)}</div>
                        <div className={css.uploadDate}>{timeAgo(file.uploadDate)}</div>
                        <div className={css.downloadIcon}></div>
                        <svg onClick={(e) => handlePopupClick(e, file)} className={css.editIcon} viewBox="0 -960 960 960"><path d="M200-200h57l391-391-57-57-391 391v57Zm-80 80v-170l528-527q12-11 26.5-17t30.5-6q16 0 31 6t26 18l55 56q12 11 17.5 26t5.5 30q0 16-5.5 30.5T817-647L290-120H120Zm640-584-56-56 56 56Zm-141 85-28-29 57 57-29-28Z"/></svg>
                        <svg onClick={(e) => handleFolderDelete(file.id, e)} className={css.deleteIcon} viewBox="0 -960 960 960"><path d="M280-120q-33 0-56.5-23.5T200-200v-520h-40v-80h200v-40h240v40h200v80h-40v520q0 33-23.5 56.5T680-120H280Zm400-600H280v520h400v-520ZM360-280h80v-360h-80v360Zm160 0h80v-360h-80v360ZM280-720v520-520Z"/></svg>
                    </div>)
                }
                if(file.uploadDate === 0) {
                    const currentProgress = filesInProgress.find((f) => f.id === file.id)
                    if(currentProgress) {
                    return (
                        <div className={`${css.filesElement}`} key={file.id}>
                            <div className={`${css.fileName} ${css.inProgress}`} title={file.path}>{file.path.split('/').pop()}</div>
                            <div className={css.upladedBytes}>Progress: {getUnitSize(currentProgress.bytesUploaded)}{getUnit(currentProgress.bytesUploaded)}/{getUnitSize(file.size)}{getUnit(file.size)}</div>
                            <div className={css.uploadSpeed}>Speed: {getUnitSize(currentProgress.uploadSpeedBytes)}{getUnit(currentProgress.uploadSpeedBytes)}/s</div>
                            <svg onClick={() => pausedFiles.current.add(file.id)} className={css.pauseIcon} viewBox="0 -960 960 960"><path d="M520-200v-560h240v560H520Zm-320 0v-560h240v560H200Zm400-80h80v-400h-80v400Zm-320 0h80v-400h-80v400Zm0-400v400-400Zm320 0v400-400Z"/></svg>
                            <svg onClick={e => handlePopupClick(e, file)} className={css.editIcon} viewBox="0 -960 960 960"><path d="M200-200h57l391-391-57-57-391 391v57Zm-80 80v-170l528-527q12-11 26.5-17t30.5-6q16 0 31 6t26 18l55 56q12 11 17.5 26t5.5 30q0 16-5.5 30.5T817-647L290-120H120Zm640-584-56-56 56 56Zm-141 85-28-29 57 57-29-28Z"/></svg>
                            <svg onClick={e => {pausedFiles.current.add(file.id); handleAbortUpload(file.id, e)}} className={css.deleteIcon} viewBox="0 -960 960 960"><path d="M280-120q-33 0-56.5-23.5T200-200v-520h-40v-80h200v-40h240v40h200v80h-40v520q0 33-23.5 56.5T680-120H280Zm400-600H280v520h400v-520ZM360-280h80v-360h-80v360Zm160 0h80v-360h-80v360ZM280-720v520-520Z"/></svg>
                        </div>
                    )}
                    return (
                        <div className={`${css.filesElement}`} key={file.id}>
                            <div className={`${css.fileName} ${css.inProgress}`} title={file.path}>{file.path.split('/').pop()}</div>
                            <div className={css.username}>{file.ownerUsername}</div>
                            <div className={css.size}>{getUnitSize(file.size)}{getUnit(file.size)}</div>
                            <div className={css.uploadDate}>In progress...</div>
                            {file.ownerUsername === username ? <svg onClick={() => handleFileResume(file.id)} className={css.downloadIcon} viewBox="0 -960 960 960"><path d="M320-200v-560l440 280-440 280Zm80-280Zm0 134 210-134-210-134v268Z"/></svg> 
                                : <svg className={css.downloadIcon} viewBox="0 -960 960 960"></svg>}
                            <svg onClick={e => handlePopupClick(e, file)} className={css.editIcon} viewBox="0 -960 960 960"><path d="M200-200h57l391-391-57-57-391 391v57Zm-80 80v-170l528-527q12-11 26.5-17t30.5-6q16 0 31 6t26 18l55 56q12 11 17.5 26t5.5 30q0 16-5.5 30.5T817-647L290-120H120Zm640-584-56-56 56 56Zm-141 85-28-29 57 57-29-28Z"/></svg>
                            <svg onClick={e => {handleAbortUpload(file.id, e)}} className={css.deleteIcon} viewBox="0 -960 960 960"><path d="M280-120q-33 0-56.5-23.5T200-200v-520h-40v-80h200v-40h240v40h200v80h-40v520q0 33-23.5 56.5T680-120H280Zm400-600H280v520h400v-520ZM360-280h80v-360h-80v360Zm160 0h80v-360h-80v360ZM280-720v520-520Z"/></svg>
                        </div>
                    )
                }
                return (
                <div className={`${css.filesElement}`} key={file.id}>
                    <div className={css.fileName} title={file.path}>{file.path.split('/').pop()}</div>
                    <div className={css.username}>{file.ownerUsername}</div>
                    <div className={css.size}>{getUnitSize(file.size)}{getUnit(file.size)}</div>
                    <div className={css.uploadDate}>{timeAgo(file.uploadDate)}</div>
                    <svg onClick={e => handleDownload(file.id, e)} className={css.downloadIcon} viewBox="0 -960 960 960"><path d="M480-320 280-520l56-58 104 104v-326h80v326l104-104 56 58-200 200ZM240-160q-33 0-56.5-23.5T160-240v-120h80v120h480v-120h80v120q0 33-23.5 56.5T720-160H240Z"/></svg>
                    <svg onClick={e => handlePopupClick(e, file)} className={css.editIcon} viewBox="0 -960 960 960"><path d="M200-200h57l391-391-57-57-391 391v57Zm-80 80v-170l528-527q12-11 26.5-17t30.5-6q16 0 31 6t26 18l55 56q12 11 17.5 26t5.5 30q0 16-5.5 30.5T817-647L290-120H120Zm640-584-56-56 56 56Zm-141 85-28-29 57 57-29-28Z"/></svg>
                    <svg onClick={e => handleFileDelete(file.id, e)} className={css.deleteIcon} viewBox="0 -960 960 960"><path d="M280-120q-33 0-56.5-23.5T200-200v-520h-40v-80h200v-40h240v40h200v80h-40v520q0 33-23.5 56.5T680-120H280Zm400-600H280v520h400v-520ZM360-280h80v-360h-80v360Zm160 0h80v-360h-80v360ZM280-720v520-520Z"/></svg>
                </div>)
            })}
        </div>
        {createFolderPopup ? <CreateFolderPopup handleCreateFolderClick={handleCreateFolderClick} folderName={folderName}
        loading={loading} handleCreateFolder={handleCreateFolder} status={status}/> : <></>}

        {warningPopup ? <WarningPopup message={warningMessage} setWarningPopup={setWarningPopup}/> : <></>}

        {fileNameChangePopup ? <FileNameChangePopup handleFileNameChange={handleFileNameChange} loading={loading} 
        setFileNameChangePopup={setFileNameChangePopup} status={status} nameChange={nameChange} 
        currentlyModifiedFile={currentlyModifiedFile} /> : <></>}

        {folderNameChangePopup ? <FolderNameChangePopup handleFolderNameChange={handleFolderNameChange} loading={loading} 
        setFolderNameChangePopup={setFolderNameChangePopup} status={status} nameChange={nameChange} 
        currentlyModifiedFile={currentlyModifiedFile} /> : <></>}
    </div>
    </div>
    )
}

function CreateFolderPopup({handleCreateFolderClick, folderName, loading, handleCreateFolder, status}: 
    {handleCreateFolderClick: () => void, folderName: React.RefObject<HTMLInputElement | null>, loading: boolean,
        handleCreateFolder: (e: React.MouseEvent<HTMLButtonElement, MouseEvent>) => void, status: string
    }) {
    return (
    <div className={css.createFolderPopupWrapper} onClick={handleCreateFolderClick}>
            <div className={css.createFolderPopup} onClick={(e) => {e.stopPropagation()}}>
                <div className={css.createFolderPopupHeader}>Create folder</div>
                <form className={css.formContainer}>
                    <input className={css.createFolderInput} ref={folderName} type="text" placeholder="Name" autoComplete="off" maxLength={100}/>
                    <div className={`${css.createFolderStatus} ${status !== "" ? css.createFolderErrorStatus : ""}`}>
                        {status === "" ? "" : status}
                    </div>
                    <button disabled={loading} className={!loading ? css.createFolderButton : css.createFolderButtonBlocked} onClick={handleCreateFolder}>
                        Create
                    </button>
                </form>
            </div>
    </div>
    )
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

function FileNameChangePopup({handleFileNameChange, loading, setFileNameChangePopup, status, nameChange, currentlyModifiedFile}: 
    {handleFileNameChange: (file: S3File, e: React.MouseEvent<HTMLButtonElement, MouseEvent>) => void, loading: boolean, 
        setFileNameChangePopup: React.Dispatch<React.SetStateAction<boolean>>, 
        status: string, nameChange: React.RefObject<HTMLInputElement | null>, currentlyModifiedFile: S3File | null
    }) {
    return (
    <div className={css.createFolderPopupWrapper} onClick={() => setFileNameChangePopup(b => !b)}>
            <div className={css.createFolderPopup} onClick={(e) => {e.stopPropagation()}}>
                <div className={css.createFolderPopupHeader}>Change file name</div>
                <form className={css.formContainer}>
                    <input className={css.createFolderInput} ref={nameChange} type="text" placeholder="Name" autoComplete="off" maxLength={100}/>
                    <div className={`${css.createFolderStatus} ${status !== "" ? css.createFolderErrorStatus : ""}`}>
                        {status === "" ? "" : status}
                    </div>
                    <button disabled={loading} className={!loading ? css.createFolderButton : css.createFolderButtonBlocked} 
                    onClick={(e) => handleFileNameChange(currentlyModifiedFile!, e)}>
                        Change
                    </button>
                </form>
            </div>
    </div>
    )
}

function FolderNameChangePopup({handleFolderNameChange, loading, setFolderNameChangePopup, status, nameChange, currentlyModifiedFile}: 
    {handleFolderNameChange: (file: S3File, e: React.MouseEvent<HTMLButtonElement, MouseEvent>) => void, loading: boolean, 
        setFolderNameChangePopup: React.Dispatch<React.SetStateAction<boolean>>, 
        status: string, nameChange: React.RefObject<HTMLInputElement | null>, currentlyModifiedFile: S3File | null
    }) {
    return (
    <div className={css.createFolderPopupWrapper} onClick={() => setFolderNameChangePopup(b => !b)}>
            <div className={css.createFolderPopup} onClick={(e) => {e.stopPropagation()}}>
                <div className={css.createFolderPopupHeader}>Change folder name</div>
                <form className={css.formContainer}>
                    <input className={css.createFolderInput} ref={nameChange} type="text" placeholder="Name" autoComplete="off" maxLength={100}/>
                    <div className={`${css.createFolderStatus} ${status !== "" ? css.createFolderErrorStatus : ""}`}>
                        {status === "" ? "" : status}
                    </div>
                    <button disabled={loading} className={!loading ? css.createFolderButton : css.createFolderButtonBlocked} 
                    onClick={(e) => handleFolderNameChange(currentlyModifiedFile!, e)}>
                        Change
                    </button>
                </form>
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

    if (seconds <= 0) return "now"
    if (seconds < 60) return `${seconds} seconds ago`;
    if (minutes < 60) return `${minutes} minutes ago`;
    if (hours < 24) return `${hours} hours ago`;
    return `${days} days ago`;
}
