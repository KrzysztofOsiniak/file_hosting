export type ErrorResponse = {
    message: string
}
export let messages: {userHasNoSpace: string, insufficientPermission: string, 
fileAlreadyExists: string, containingFolderDoesNotExist: string} = {
    userHasNoSpace: "User does not have enough space",
    insufficientPermission: "User has insufficient permission",
    fileAlreadyExists: "File already exists",
    containingFolderDoesNotExist: "Containing folder does not exist"
}

export type RepositoryResponse = {
    name: string,
    members: Member[],
    files: S3File[],
    userPermission: "owner" | "none" |"full" | "read",
    ownerUsername: string,
    visibility: "public" | "private",
}
export type Repository = {
    name: string,
    userPermission: "owner" | "none" |"full" | "read",
    ownerUsername: string,
    visibility: "public" | "private",
}
type Member = {
    id: number,
    username: string,
    permission: "" | "full" | "read"
}
export type S3File = {
    id: number,
    ownerUsername: string,
    path: string,
    type: "file" | "folder",
    size: number,
    uploadDate: number
}
export type FileInProgress = {
    id: number,
    bytesUploaded: number,
    bytesUploadedHidden: number, // Bytes that are not displayed, updated more often.
    bytesUploadedPrevious: number,
    uploadSpeedBytes: number,
    timeFromLastUploadedBytes: Date,
    error: string,
}
