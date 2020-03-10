interface ReponseErrorMessage {
    code: number
    message: string
}

export interface ResponseMessage {
    result?: any
    error?: ReponseErrorMessage
    id: string
}

export enum Method {
    SUBSCRIBE = "subscribe",
    UNSUBSCRIBE = "unsubscribe"
}