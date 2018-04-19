import * as H from 'history'
import * as React from 'react'
import * as GQL from '../../backend/graphqlschema'
import { QueryButton } from './QueryButton'

interface Props {
    className: string
    title: string
    description?: string
    errorBody?: React.ReactFragment
    proposedQueries?: GQL.ISearchQueryDescription[]

    location: H.Location
}

export class SearchAlert extends React.Component<Props, {}> {
    public render(): JSX.Element | null {
        return (
            <div className={`search-alert ${this.props.className}`}>
                <div className="search-alert__message">
                    {<h2 className="search-alert__title">{this.props.title}</h2>}
                    {this.props.description && <p className="search-alert__description">{this.props.description}</p>}
                </div>
                {this.props.errorBody}
                {this.props.proposedQueries && (
                    <ul className="search-alert__proposed-queries">
                        {this.props.proposedQueries.map((proposedQuery, i) => (
                            <li key={i} className="search-alert__proposed-query">
                                <span className="search-alert__proposed-query-did-you-mean">Did you mean: </span>
                                <QueryButton query={proposedQuery.query} />
                                <span className="search-alert__proposed-query-description">
                                    {proposedQuery.description && ` — ${proposedQuery.description}`}
                                </span>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        )
    }
}