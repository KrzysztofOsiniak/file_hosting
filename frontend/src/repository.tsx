import { useLoaderData, useOutletContext } from "react-router-dom"
import css from './css/repository.module.scss'
import type { ErrorResponse, RepositoryResponse } from "./types"
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
    const repositoryID = useLoaderData()
    const {setHomePage, setFreeSpace, username} = useOutletContext<{setHomePage: React.Dispatch<React.SetStateAction<boolean>>, 
        setFreeSpace: React.Dispatch<React.SetStateAction<number>>, username: string}>()

    const [repository, setRepository] = useState<Repository | number | null>(null)
    const [files, setFiles] = useState<S3File[] | null>(null)
    const [displayFiles, setDisplayFiles] = useState<S3File[] | null>(null)
    const [currentPath, setCurrentPath] = useState("")
    const [createFolderPopup, setCreateFolderPopup] = useState(false)
    const [status, setStatus] = useState("")
    const [loading, setLoading] = useState(false)
    const [warningPopup, setWarningPopup] = useState(false)
    const [warningMessage, setWarningMessage] = useState("")

    const folderName = useRef<HTMLInputElement>(null)
    const fileInputRef = useRef<HTMLInputElement>(null)

    function handleFileUpload() {
        fileInputRef.current?.click()
    }

    async function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
        if(e.target.files === null) return
        const file = e.target.files[0]
        const path = currentPath
        const res = await fetch('/api/file/upload-start', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                key: path + file.name, size: file.size, repositoryID: parseInt(repositoryID)
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
                setWarningMessage("unknown error")
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
            if(res.status !== 200) {
                setWarningMessage(`Error uploading the file, status: ${res.status}`);
                setWarningPopup(true); 
                return
            }
                
            const eTag = res.headers.get("ETag")
            if (eTag === null) {
                setWarningMessage(`Could not read the response headers when uploading the file`);
                setWarningPopup(true); 
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
                setWarningPopup(true); 
                return
            }
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
        if(resComplete.status !== 200) {
            if(resComplete.status === 403) {
                setWarningMessage(`This file was deleted`)
            } else {
                setWarningMessage(`Unknown error`)
            }
            setWarningPopup(true); 
            return
        }
        const completeData = await resComplete.json()
        setFiles(f => f!.map((v, _) => {
            if(v.id === fileID) {
                return {
                    id: v.id,
                    ownerUsername: v.ownerUsername,
                    path: v.path,
                    size: v.size,
                    type: v.type,
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
                key: currentPath + name, repositoryID: parseInt(repositoryID)
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
            setStatus("You already have a folder with that name")
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

    async function handleFileNameChange(e: React.MouseEvent<SVGSVGElement>) {
        e.stopPropagation()
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
        fetch(`/api/repository/${repositoryID}`)
        .then((res): Promise<RepositoryResponse> => {
            if(res.status != 200) {
                setRepository(res.status)
                throw new Error()
            }
            return res.json()
        })
        .then(data => {
            setRepository({name: data.name, userPermission: data.userPermission})
            setFiles(data.files)
        })
        .catch()
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

    if(repository === 404 || repository === 401) {
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
        <div className={css.repositoryTitle}>{repository.name}</div>
        <div className={css.filesContainer}>
            <input type="file" onChange={handleFileChange} ref={fileInputRef} style={{display: 'none'}}/>
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
                }).map(file => {
                if(file.type === "folder") {
                    return (
                    <div onClick={() => handleFolderClick(file.path + "/")} className={`${css.filesElement} ${css.selectable}`} key={file.id}>
                        <div className={`${css.fileName} ${css.folderElement}`} title={file.path}>{file.path.split('/').pop()}</div>
                        <div className={css.username}>{file.ownerUsername}</div>
                        <div className={css.size}>{getUnitSize(file.size)}{getUnit(file.size)}</div>
                        <div className={css.uploadDate}>{timeAgo(file.uploadDate)}</div>
                        <div className={css.downloadIcon}></div>
                        <svg onClick={e => handleFileNameChange(e)} className={css.editIcon} viewBox="0 -960 960 960"><path d="M200-200h57l391-391-57-57-391 391v57Zm-80 80v-170l528-527q12-11 26.5-17t30.5-6q16 0 31 6t26 18l55 56q12 11 17.5 26t5.5 30q0 16-5.5 30.5T817-647L290-120H120Zm640-584-56-56 56 56Zm-141 85-28-29 57 57-29-28Z"/></svg>
                        <svg onClick={(e) => handleFolderDelete(file.id, e)} className={css.deleteIcon} viewBox="0 -960 960 960"><path d="M280-120q-33 0-56.5-23.5T200-200v-520h-40v-80h200v-40h240v40h200v80h-40v520q0 33-23.5 56.5T680-120H280Zm400-600H280v520h400v-520ZM360-280h80v-360h-80v360Zm160 0h80v-360h-80v360ZM280-720v520-520Z"/></svg>
                    </div>)
                }
                if(file.uploadDate === 0) {
                    return (
                    <div className={`${css.filesElement} ${css.inProgress}`} key={file.id}>
                        <div className={css.fileName}>{file.path.split('/').pop()}</div>
                    </div>)
                }
                return (
                <div className={`${css.filesElement}`} key={file.id}>
                    <div className={css.fileName} title={file.path}>{file.path.split('/').pop()}</div>
                    <div className={css.username}>{file.ownerUsername}</div>
                    <div className={css.size}>{getUnitSize(file.size)}{getUnit(file.size)}</div>
                    <div className={css.uploadDate}>{timeAgo(file.uploadDate)}</div>
                    <svg onClick={e => handleDownload(file.id, e)} className={css.downloadIcon} viewBox="0 -960 960 960"><path d="M480-320 280-520l56-58 104 104v-326h80v326l104-104 56 58-200 200ZM240-160q-33 0-56.5-23.5T160-240v-120h80v120h480v-120h80v120q0 33-23.5 56.5T720-160H240Z"/></svg>
                    <svg onClick={e => handleFileNameChange(e)} className={css.editIcon} viewBox="0 -960 960 960"><path d="M200-200h57l391-391-57-57-391 391v57Zm-80 80v-170l528-527q12-11 26.5-17t30.5-6q16 0 31 6t26 18l55 56q12 11 17.5 26t5.5 30q0 16-5.5 30.5T817-647L290-120H120Zm640-584-56-56 56 56Zm-141 85-28-29 57 57-29-28Z"/></svg>
                    <svg onClick={e => handleFileDelete(file.id, e)} className={css.deleteIcon} viewBox="0 -960 960 960"><path d="M280-120q-33 0-56.5-23.5T200-200v-520h-40v-80h200v-40h240v40h200v80h-40v520q0 33-23.5 56.5T680-120H280Zm400-600H280v520h400v-520ZM360-280h80v-360h-80v360Zm160 0h80v-360h-80v360ZM280-720v520-520Z"/></svg>
                </div>)
            })}
        </div>
        {createFolderPopup ? <CreateFolderPopup handleCreateFolderClick={handleCreateFolderClick} folderName={folderName}
        loading={loading} handleCreateFolder={handleCreateFolder} status={status}/> : <></>}
        {warningPopup ? <WarningPopup message={warningMessage} setWarningPopup={setWarningPopup}/> : <></>}
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
