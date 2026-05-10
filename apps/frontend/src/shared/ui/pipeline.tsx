import Link from "next/link";

export interface PipelineStep {
  label: string;
  detail: string;
  href?: string;
}

export function Pipeline({ steps }: { steps: PipelineStep[] }) {
  return (
    <div className="pipeline">
      {steps.map((step, index) => {
        const content = (
          <>
            <span className="pipeline-index">{index + 1}</span>
            <strong>{step.label}</strong>
            <small>{step.detail}</small>
          </>
        );

        return step.href ? (
          <Link className="pipeline-step" href={step.href} key={step.label}>
            {content}
          </Link>
        ) : (
          <div className="pipeline-step" key={step.label}>
            {content}
          </div>
        );
      })}
    </div>
  );
}
